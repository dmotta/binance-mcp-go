# Design: Bugfix Binance MCP Server (BUG-001, BUG-002)

> **Spec type:** Bugfix (Design Phase)
> **Fuente:** [bugfix.md](bugfix.md) — validado contra Binance Testnet
> **Baseline analizado:** commit `0ab1db4` (todas las referencias `file:line` apuntan a ese commit)
> **Fecha:** 2026-06-09
> *Nota: este archivo reemplaza un stub previo de componentes generales; ver spec.md para la arquitectura global.*

---

## Arquitectura relevante

El servidor sigue un patrón ports & adapters:

- `internal/tools/*.go` — handlers MCP: parsean argumentos JSON y delegan en el port.
- `internal/port/port.go` — interfaz `BinancePort` y structs de parámetros (frontera tools↔adapter).
- `internal/adapter/adapter.go` — implementación sobre el SDK `go-binance/v2`. El struct
  `BinanceAdapter` (adapter.go:17-21) mantiene **tres clientes HTTP independientes**, uno
  por mercado, construidos en `main.go`:

  | Cliente | Tipo SDK | BaseURL (testnet) | Endpoint REST |
  |---------|----------|-------------------|---------------|
  | `a.spot` | `binance.Client` | `testnet.binance.vision` | `/api/v3/*` |
  | `a.futures` | `futures.Client` | `testnet.binancefuture.com` | `/fapi/v1/*` |
  | `a.opts` | `options.Client` | (sin testnet) | `/eapi/v1/*` |

  El ruteo de mercado queda **fijado en tiempo de compilación** por cuál cliente usa cada
  método del adapter. No hay ruteo dinámico.

---

# BUG-001: `create_trailing_stop_order` — `callbackRate` → `trailingDelta`

## Root cause

**No existe ninguna conversión ×100 en el código.** La cadena de mapeo en `0ab1db4` es una
identidad textual:

1. `internal/tools/risk_control.go:106` — el handler toma el número JSON y lo formatea como
   string: `CallbackRate: fmt.Sprintf("%g", getFloat(req, "callbackRate"))` → `1.5` se
   convierte en `"1.5"`.
2. `internal/port/port.go:88` — `TrailingStopOrderParams.CallbackRate string` transporta el
   valor sin tipo semántico (ni porcentaje ni BIPS).
3. `internal/adapter/adapter.go:208` — `TrailingDelta(p.CallbackRate)` pasa el string crudo
   al SDK, que lo envía literal como form param `trailingDelta=1.5`. Binance lo rechaza con
   `-1100` porque su regex es `^[0-9]{1,20}$` (solo dígitos enteros).

**Por qué parece no determinístico:** el código es 100 % determinístico (identidad). La
variación entre "sesiones" observada en la evidencia del spec no proviene de estado del
servidor — no hay inicialización condicional, type assertion variable ni estado compartido
entre llamadas — sino del **cliente LLM que invoca la herramienta**. Como el schema
(`risk_control.go:98`) declara `callbackRate` solo como `{"type":"number"}` sin documentar
unidad ni rango, cada sesión del modelo interpreta la unidad a su criterio: unas pasan
porcentaje (`1.5` → `-1100` regex fail) y otras pre-convierten a BIPS (`150` → la orden pasa
"raw", y si además el valor real deseado era 1.5 %, queda una orden con delta incorrecto, o
falla `-1013` si excede el filtro). La "conversión ×100 que a veces se aplica" vive en el
prompt del cliente, no en este repo. El defecto de código es la **ambigüedad del contrato**:
unidad sin especificar, sin validación, y passthrough textual.

## Fix approach

Hacer el contrato explícito y la conversión determinística, validando antes de llamar al API:

1. **Tipo:** `CallbackRate` pasa de `string` a `float64` en el port (porcentaje, semántica
   única). El handler deja de formatear con `%g` y pasa el número.
2. **Conversión (adapter):** `bips := int(math.Round(callbackRate * 100))` y
   `TrailingDelta(strconv.Itoa(bips))` — siempre entero, siempre ×100, cumple
   `^[0-9]{1,20}$`.
3. **Validación (adapter, antes del API call):** rechazar `callbackRate < 0.1 || > 20` con
   `"callbackRate must be between 0.1 and 20 (percent), got %g"`. El rango 0.1–20 % cubre
   10–2000 BIPS, el máximo del filtro `TRAILING_DELTA` evidenciado en el spec (20 → OK,
   21 → `-1013`).
4. **Schema (tools):** documentar `callbackRate` con `description` (unidad = porcentaje,
   rango 0.1–20) y `minimum`/`maximum` JSON Schema, para que el cliente LLM no tenga que
   adivinar la unidad — esto ataca la causa raíz del no-determinismo observado.

La validación se coloca en el adapter (junto a la conversión) porque es la última frontera
antes del SDK: cualquier caller futuro del port queda protegido, no solo el handler MCP.

---

# BUG-002: gestión de órdenes solo opera contra Spot

## Root cause

Las cuatro herramientas de gestión usan exclusivamente el cliente spot, mientras las de
creación/consulta de futuros usan `a.futures`. En `0ab1db4`:

| Herramienta | Adapter | Cliente | Endpoint efectivo |
|-------------|---------|---------|-------------------|
| `cancel_order` | adapter.go:51 → `a.spot.NewCancelOrderService()` (adapter.go:52) | spot | `DELETE /api/v3/order` |
| `get_open_orders` | adapter.go:70 → `a.spot.NewListOpenOrdersService()` (adapter.go:71) | spot | `GET /api/v3/openOrders` |
| `get_order_status` | adapter.go:94 → `a.spot.NewGetOrderService()` (adapter.go:95) | spot | `GET /api/v3/order` |
| `cancel_all_orders` | adapter.go:136 → `a.spot.NewCancelOpenOrdersService()` (adapter.go:137) | spot | `DELETE /api/v3/openOrders` |
| **contraste:** `create_contract_order` | adapter.go:426 → `a.futures.NewCreateOrderService()` (adapter.go:427) | futures | `POST /fapi/v1/order` |
| **contraste:** `get_futures_positions` | adapter.go:503 → `a.futures.NewGetPositionRiskService()` (adapter.go:504) | futures | `GET /fapi/v2/positionRisk` |

Un `orderId` de futuros (p. ej. `14615025460`) no existe en el order book spot, de ahí
`-2011 Unknown order sent` / `-2013 Order does not exist` / `[]`. Los structs de parámetros
(`port.go:25-46`) no tienen campo de mercado, así que el adapter no tiene información para
rutear aunque quisiera.

## Fix approach — evaluación de los 3 candidatos del spec

1. **Parámetro `market` (elegido).** Añadir `market: "spot" | "futures"` (default `"spot"`)
   a las cuatro herramientas; el adapter rutea al cliente correspondiente.
   - ✔ Retrocompatible: sin `market` el comportamiento es idéntico al actual (spot).
   - ✔ Consistente con la arquitectura existente: reutiliza el cliente `a.futures` ya
     construido y los servicios del SDK (`futures.NewCancelOrderService`, etc.) con la misma
     forma de respuesta (`OrderID/Symbol/Status/Side/Type/Price/OrigQuantity`).
   - ✔ Mantiene 1 herramienta = 1 operación de negocio; el LLM cliente decide el mercado,
     que es quien sabe dónde creó la orden.
2. **Herramientas dedicadas (`cancel_futures_order`, …).** Descartado: duplica 4 tools y
   sus schemas/handlers/tests (~8 registros nuevos), infla el catálogo que el LLM debe
   discriminar, y la simetría con `create_contract_order` es superficial — esa tool existe
   porque los *parámetros de creación* difieren entre mercados; en cancelación/consulta los
   parámetros son idénticos (`symbol` + `orderId`), así que la dimensión mercado es un
   atributo, no una operación distinta.
3. **Ruteo automático por rango de orderId / intento en ambos endpoints.** Descartado, como
   recomienda el propio spec: los orderIds de spot y futuros son secuencias independientes
   por símbolo sin rango disjunto garantizado por contrato; adivinar el mercado en una
   operación de cancelación arriesga actuar sobre la orden equivocada. "Probar ambos"
   duplica latencia y vuelve ambiguos los errores.

Complemento (Expected Behavior #4 del spec): los mensajes de error de las cuatro tools
incluyen el mercado consultado (`cancel_order failed (market=spot): …`), de modo que un
`-2011` deje claro contra qué book se buscó la orden.

---

## Propiedades a testear (Kiro)

Los tests unitarios mockean el transporte HTTP del SDK (un `http.RoundTripper` que captura
la request y responde JSON enlatado) — no tocan el testnet. Esto permite afirmar sobre el
**wire format real** (path del endpoint y form params), que es donde viven ambos bugs.

### BUG-001
1. **Reproducible:** con el código de `0ab1db4`, `callbackRate=1.5` produce
   `trailingDelta=1.5` en el form body (falla la aserción `trailingDelta=150`). El test
   `TestCreateTrailingStopOrder_BipsConversion` codifica la aserción correcta y por tanto
   falla contra el original.
2. **Corregido:** tabla de evidencia del spec — `1.5→150`, `1.0→100`, `0.5→50`, `5→500`,
   `10→1000`, `20→2000` enviados al API; `21`, `150`, `0`, `-1` rechazados con error de
   validación **sin emitir request**. Property-based: ∀ r ∈ [0.1, 20],
   `trailingDelta = round(r·100)` ∈ [10, 2000] y cumple `^[0-9]{1,20}$`.
3. **Sin regresión spot:** la orden sigue siendo `POST /api/v3/order` tipo `STOP_LOSS`;
   las demás tools de creación (`create_spot_order`, `create_stop_loss_order`,
   `create_take_profit_order`, `create_stop_limit_order`, `create_oco_order`) no se tocan
   (sus tests existentes en `tools_test.go` siguen en verde).

### BUG-002
1. **Reproducible:** con `0ab1db4`, `CancelOrder` siempre emite `DELETE /api/v3/order`
   (falla la aserción `/fapi/v1/order` para `market=futures`).
2. **Corregido:** con `market=futures`, las cuatro operaciones golpean
   `/fapi/v1/order` (DELETE), `/fapi/v1/allOpenOrders` (DELETE), `/fapi/v1/order` (GET),
   `/fapi/v1/openOrders` (GET).
3. **Sin regresión spot:** con `market` omitido o `"spot"`, las cuatro operaciones siguen
   golpeando `/api/v3/order`, `/api/v3/openOrders` con los mismos parámetros, y las tools
   de futuros intactas (`create_contract_order`, `get_futures_positions`, `get_positions`,
   `close_position`, `set_leverage`, `set_margin_mode`) no se modifican.

## Out of scope

Conforme al spec: opciones (`create_option_order`, `get_option_*`) y `transfer_funds` no se
tocan.
