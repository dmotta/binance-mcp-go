# Bugfix Spec: Binance MCP Server

> **Spec type:** Bugfix (Analysis Phase)
> **Component:** Binance-MCP-Server
> **Environment validado:** Binance Testnet (`environment: "testnet"`, version `1.0.0`)
> **Estructura:** [Kiro Bugfix Specs](https://kiro.dev/docs/specs/bugfix-specs/) — formato EARS
> **Fecha:** 2026-06-09
> **Estado:** Analysis (pendiente Design + Tasks)

---

## Resumen

Se identificaron y validaron empíricamente dos defectos en el Binance MCP Server durante un test sistemático de las 21 interfaces expuestas. Ambos bugs fueron reproducidos con múltiples llamadas contra el entorno testnet.

| ID | Severidad | Interfaz afectada | Naturaleza |
|----|-----------|-------------------|------------|
| BUG-001 | Media | `create_trailing_stop_order` | Conversión de unidad no determinística en `callbackRate` |
| BUG-002 | Alta | `cancel_order`, `cancel_all_orders`, `get_order_status`, `get_open_orders` | Gestión de órdenes solo cubre el endpoint Spot; las órdenes de futuros quedan huérfanas |

---

# BUG-001: `create_trailing_stop_order` — conversión `callbackRate` → `trailingDelta` no determinística

## Contexto

El parámetro público de la herramienta es `callbackRate` (tipo `number`), cuyo nombre sugiere un porcentaje (ej. `1.5` = 1.5%). Internamente la API de Binance espera el campo `trailingDelta`, expresado en **BIPS** (basis points enteros: `1.5%` = `150` BIPS), con validación regex `^[0-9]{1,20}$` y un filtro de rango máximo.

El defecto es que la conversión `callbackRate → trailingDelta` (multiplicación ×100) **no se aplica de forma consistente** entre inicializaciones de la herramienta.

## Current Behavior (Defect)

- WHEN se llama `create_trailing_stop_order` con `callbackRate=1.5` en una sesión donde la conversión ×100 NO se aplica, THEN el sistema envía `trailingDelta=1.5` a la API y falla con `code=-1100, msg=Illegal characters found in parameter 'trailingDelta'; legal range is '^[0-9]{1,20}$'`.
- WHEN se llama `create_trailing_stop_order` con `callbackRate=150` en una sesión donde la conversión ×100 SÍ se aplica, THEN el sistema envía `trailingDelta=15000` (150%) y falla con `code=-1013, msg=Filter failure: TRAILING_DELTA` por exceder el máximo permitido.
- WHEN el mismo valor de `callbackRate` se envía en dos sesiones distintas de carga de la herramienta, THEN el sistema produce resultados contradictorios (éxito en una, fallo en la otra).

### Evidencia de reproducción

| Valor `callbackRate` | Sesión 1 (sin ×100) | Sesión 2 (con ×100) | trailingDelta resultante (S2) |
|----------------------|---------------------|---------------------|-------------------------------|
| `1.5` | ❌ `-1100` regex fail | ✅ NEW | 150 BIPS (1.5%) |
| `1.0` | — | ✅ NEW | 100 BIPS (1.0%) |
| `0.5` | — | ✅ NEW | 50 BIPS (0.5%) |
| `5`   | — | ✅ NEW | 500 BIPS (5%) |
| `10`  | — | ✅ NEW | 1000 BIPS (10%) |
| `20`  | — | ✅ NEW | 2000 BIPS (20%, máximo) |
| `21`  | — | ❌ `-1013` filter fail | 2100 BIPS (excede máx) |
| `150` | ✅ NEW (raw) | ❌ `-1013` filter fail | 15000 BIPS (excede máx) |

## Expected Behavior (Correct)

- WHEN se llama `create_trailing_stop_order` con `callbackRate=1.5`, THEN el sistema SHALL convertir consistentemente a `trailingDelta=150` BIPS y crear la orden, independientemente de la sesión.
- WHEN se llama `create_trailing_stop_order` con un `callbackRate` fuera del rango válido (≤ 0 o > 20), THEN el sistema SHALL rechazar la entrada con un mensaje de validación claro ANTES de llamar a la API (ej. `"callbackRate must be between 0.1 and 20 (percent)"`).
- WHEN se documenta el parámetro `callbackRate`, THEN la descripción SHALL especificar explícitamente que la unidad es porcentaje y el rango aceptado es `0.1`–`20`.

## Unchanged Behavior (Regression Prevention)

- WHEN se llama `create_trailing_stop_order` con `quantity`, `side` y `symbol` válidos, THEN el sistema SHALL CONTINUAR creando órdenes de tipo `STOP_LOSS` con el `orderId` retornado.
- WHEN se llama cualquier otra herramienta de creación de órdenes (`create_spot_order`, `create_stop_loss_order`, `create_take_profit_order`, `create_stop_limit_order`, `create_oco_order`), THEN el sistema SHALL CONTINUAR funcionando sin cambios en su comportamiento de parámetros.
- WHEN se cancela una orden trailing stop creada exitosamente vía `cancel_order` en spot, THEN el sistema SHALL CONTINUAR retornando `status: CANCELED`.

---

# BUG-002: gestión de órdenes solo opera contra el endpoint Spot — órdenes de futuros huérfanas

## Contexto

La herramienta `create_contract_order` crea órdenes contra el endpoint de **Futuros** de Binance (orderIds del orden de `14_000_000_000+`). Sin embargo, las herramientas de consulta y cancelación de órdenes (`cancel_order`, `cancel_all_orders`, `get_order_status`, `get_open_orders`) apuntan exclusivamente al endpoint **Spot** (orderIds del orden de `2_700_000+`).

El resultado es una **asimetría crítica**: se pueden crear órdenes LIMIT de futuros pendientes pero no existe forma de consultarlas ni cancelarlas a través del MCP server, dejándolas huérfanas.

> **Nota:** este bug resultó más severo que el reporte inicial, que solo mencionaba `cancel_order`. La validación confirmó que las cuatro herramientas de gestión de órdenes comparten el mismo defecto.

## Current Behavior (Defect)

- WHEN se crea una orden LIMIT de futuros con `create_contract_order` (ej. orderId `14615025460`) y luego se intenta cancelarla con `cancel_order`, THEN el sistema falla con `code=-2011, msg=Unknown order sent`.
- WHEN existe una orden de futuros pendiente y se llama `cancel_all_orders` con el mismo símbolo, THEN el sistema falla con `code=-2011, msg=Unknown order sent` (no encuentra la orden en el libro spot).
- WHEN se consulta el estado de una orden de futuros con `get_order_status`, THEN el sistema falla con `code=-2013, msg=Order does not exist`.
- WHEN se listan órdenes abiertas con `get_open_orders` existiendo órdenes de futuros pendientes, THEN el sistema retorna `[]` (lista vacía, ignorando las órdenes de futuros).
- WHEN se intenta `close_position` para liberar una orden LIMIT de futuros que aún no se ha llenado, THEN el sistema falla con `no open position for {symbol}` (porque `close_position` solo cierra posiciones FILLED, no cancela órdenes pendientes).

### Evidencia de reproducción

Orden de prueba: `create_contract_order(symbol=BTCUSDT, side=BUY, type=LIMIT, price=50000, quantity=0.001)` → `orderId: 14615025460, status: NEW`.

| Herramienta | Resultado | Error |
|-------------|-----------|-------|
| `cancel_order` | ❌ FAIL | `code=-2011 Unknown order sent` |
| `cancel_all_orders` | ❌ FAIL | `code=-2011 Unknown order sent` |
| `get_order_status` | ❌ FAIL | `code=-2013 Order does not exist` |
| `get_open_orders` | ❌ FAIL | `[]` (no ve futuros) |
| `close_position` | ⚠️ N/A | `no open position` (no cancela órdenes pendientes) |

## Expected Behavior (Correct)

- WHEN se crea una orden de futuros con `create_contract_order`, THEN el sistema SHALL permitir cancelarla vía `cancel_order` usando el orderId y símbolo, ruteando la petición al endpoint de Futuros.
- WHEN se llama `cancel_all_orders` y existen órdenes de futuros pendientes para el símbolo, THEN el sistema SHALL cancelarlas (o exponer un parámetro/herramienta explícita para el mercado de futuros).
- WHEN se consulta `get_order_status` o `get_open_orders` para una orden de futuros, THEN el sistema SHALL retornar la orden con su estado real desde el endpoint de Futuros.
- WHEN una herramienta de gestión de órdenes recibe un orderId que no pertenece al mercado consultado, THEN el sistema SHALL indicar claramente el mercado esperado en lugar de devolver un genérico `Unknown order sent`.

### Enfoques de diseño candidatos (a detallar en `design.md`)

1. **Parámetro de mercado:** añadir `market: "SPOT" | "FUTURES"` (default `SPOT`) a las cuatro herramientas de gestión.
2. **Herramientas dedicadas:** crear `cancel_futures_order`, `get_futures_open_orders`, etc. (simétrico con `create_contract_order` / `get_futures_positions`).
3. **Ruteo automático:** inferir el mercado por rango de orderId o intentar ambos endpoints (frágil, no recomendado como solución única).

## Unchanged Behavior (Regression Prevention)

- WHEN se cancela una orden SPOT con `cancel_order` (ej. orderId `2771618`), THEN el sistema SHALL CONTINUAR retornando `status: CANCELED` con los detalles de la orden spot.
- WHEN se llama `cancel_all_orders` para un símbolo con órdenes SPOT abiertas, THEN el sistema SHALL CONTINUAR cancelándolas y retornando el arreglo de órdenes canceladas.
- WHEN se consulta `get_order_status` / `get_open_orders` para órdenes SPOT, THEN el sistema SHALL CONTINUAR retornando los datos correctos del libro spot.
- WHEN se usan las herramientas de futuros que ya funcionan (`create_contract_order`, `get_futures_positions`, `get_positions`, `close_position`, `set_leverage`, `set_margin_mode`), THEN el sistema SHALL CONTINUAR operando sin cambios.

---

## Out of Scope

Los siguientes hallazgos del test NO son bugs del código del MCP server, sino limitaciones del entorno testnet de Binance, y quedan fuera de este bugfix:

- `create_option_order`, `get_option_chain`, `get_option_info`, `get_option_positions` — Binance no ofrece testnet público de opciones (`set BINANCE_TESTNET=false`).
- `transfer_funds` — el endpoint de transferencias entre cuentas retorna HTML en lugar de JSON en testnet (`invalid character '<'`); validar contra producción antes de clasificar como bug de código.

---

## Próximos pasos (workflow Kiro)

1. **Design Phase** → `design.md`: root cause analysis del código fuente (revisar el mapeo de parámetros en el handler de `create_trailing_stop_order` y el ruteo de endpoint en las herramientas de gestión de órdenes), enfoque de fix y propiedades a testear.
2. **Tasks Phase** → tareas de implementación con property-based tests que validen: (a) el bug es reproducible, (b) el bug queda corregido, (c) no se introducen regresiones en spot.
