BUG CRÍTICO: el conector no puede colocar stop-loss de FUTUROS — dos fallas verificadas empíricamente el 2026-06-12 contra el futures testnet (testnet.binancefuture.com), con posición real de prueba.

Falla 1 — endpoint regular rechaza condicionales:
create_contract_order con type=STOP_MARKET retorna error -4120 "Order type not supported for this endpoint — use Algo Order API". Probado con closePosition=true y también con reduceOnly+quantity: ambas variantes fallan. El testnet de futuros migró las órdenes condicionales (STOP_MARKET / TAKE_PROFIT_MARKET) a la Algo Order API y el endpoint clásico ya no las acepta.

Falla 2 — las herramientas "dedicadas" son SPOT por error:
create_stop_loss_order y create_take_profit_order (registradas en internal/tools/risk_control.go) llaman a.spot.NewCreateOrderService() en internal/adapter/adapter.go líneas ~271 y ~285 → colocan órdenes SPOT silenciosamente. Verificado: retornaron orderIds 4106455/4106801 que resultaron ser órdenes spot tipo STOP_LOSS/TAKE_PROFIT (invisibles en get_open_orders(market=futures), error -2013 en get_order_status(market=futures), canceladas exitosamente con market=spot).

Trabajo requerido:

Implementar órdenes condicionales de futuros vía la Algo Order API que indica el -4120 (investigar endpoint correcto en la doc de Binance USDⓈ-M futures). Mínimo necesario: STOP_MARKET y TAKE_PROFIT_MARKET con soporte de reduceOnly y closePosition.
Agregar herramientas MCP para listar y cancelar órdenes algo — sin esto el cliente no puede verificar que sus SL/TP siguen vivos ni limpiarlos (get_open_orders y cancel_order solo ven el espacio regular de órdenes).
Resolver el engaño de nombres: create_stop_loss_order/create_take_profit_order deben renombrarse con prefijo spot_ o documentarse explícitamente como spot en su descripción.
Probar contra el futures testnet con una posición mínima real (0.002 BTCUSDT ≈ 127 USDT notional): abrir MARKET → colocar SL algo con reduceOnly → verificar que se puede listar → cancelar → cerrar posición → verificar qty residual 0. Las keys de futures testnet están en ../trading-bot-claude/.env.testnet (BINANCE_FUTURES_API_KEY/SECRET).
Contexto e impacto: el bot de trading del proyecto hermano (trading-bot-claude) está en una sesión testnet de 28 días (Etapa C) con las entradas bloqueadas (bot/ENTRIES_BLOCKED) porque sin SL colocable no debe abrir posiciones. Al terminar el fix: compilar el binario (binance-mcp-go en la raíz del repo es el que carga la app), avisar al operador que reinicie la app de Claude (el MCP se recarga al iniciar sesión) y que la sesión principal del bot re-verificará con una posición de prueba antes de desbloquear entradas.