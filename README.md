# Binance MCP Server (Golang)

[![Go Version](https://img.shields.io/github/go-mod/go-version/dmotta/binance-mcp-go)](https://golang.org)
[![MCP Protocol](https://img.shields.io/badge/mcp-protocol-orange)](https://modelcontextprotocol.io)

Un servidor del **Model Context Protocol (MCP)** robusto, seguro y altamente configurable que expone las APIs de Binance (Spot, Futures, Options) como herramientas interactivas directamente consumibles por modelos de lenguaje (LLMs) y clientes MCP compatibles (por ejemplo, Claude Desktop).

Desarrollado bajo una **Arquitectura Hexagonal (Puertos y Adaptadores)** en Go, el servidor implementa telemetría avanzada (OpenTelemetry), limitación de tasa inteligente, disyuntor (circuit breaker) y reintentos automatizados para garantizar una integración premium y libre de fallos con Binance.

---

## 🗺️ Arquitectura del Sistema

El proyecto está diseñado bajo los principios de la Arquitectura Hexagonal para desacoplar el protocolo MCP de las APIs externas de Binance:

```mermaid
graph TD
    subgraph Cliente MCP
        LLM[Claude / LLM Engine] -->|Invoca herramienta| MCPServer[MCP Server stdio]
    end

    subgraph Capa Interna (Módulos Go)
        MCPServer -->|Mapea JSON-RPC| Tools[Módulo: internal/tools]
        Tools -->|Usa interfaces| Port[Puerto: internal/port.BinancePort]
        Adapter[Adaptador: internal/adapter] -.->|Implementa| Port
    end

    subgraph Middleware HTTP & Clientes SDK
        Adapter -->|Configura HTTP Client| Httpmw[Middleware Chain: internal/httpmw]
        Httpmw -->|OTel / CB / Retry / RateLimit| HttpClient[http.Client]
        HttpClient -->|Llamadas API| BinanceSDK[go-binance SDK Client]
    end

    subgraph API Externa
        BinanceSDK -->|Spot API| SpotEnd[Binance Spot Endpoint]
        BinanceSDK -->|Futures API| FutEnd[Binance Futures Endpoint]
        BinanceSDK -->|Options API| OptEnd[Binance Options Endpoint]
    end

    classDef core fill:#e1f5fe,stroke:#01579b,stroke-width:2px;
    classDef adapter fill:#e8f5e9,stroke:#1b5e20,stroke-width:2px;
    classDef client fill:#fff3e0,stroke:#e65100,stroke-width:2px;
    class Port core;
    class Adapter adapter;
    class Httpmw client;
```

Para una explicación exhaustiva de cada línea de código y patrón utilizado, consulta la [Documentación Detallada de Arquitectura](file:///Users/dmotta/Documents/ws-github-dmotta/binance-mcp-go/docs/architecture.md).

---

## ✨ Características Principales

*   **Hexagonal Clean Architecture**: Desacoplamiento estricto a través de la interfaz `BinancePort`. Facilidad de pruebas unitarias mediante mocks generados.
*   **Resiliencia HTTP de Grado Industrial**:
    *   **Rate Limiting Dinámico**: Intercepta cabeceras `X-Mbx-Used-Weight-1M` de Binance y pausa preventivamente antes de recibir ban-limits (HTTP 418 / 429).
    *   **Circuit Breaker (Disyuntor)**: Impide peticiones innecesarias tras 5 fallos consecutivos del servidor usando `gobreaker`.
    *   **Políticas de Reintento**: Estrategia de reintento exponencial (`backoff`) con protección de idempotencia (no reintenta peticiones de trading con cuerpos que no se puedan rebobinar).
    *   **Observabilidad Completa**: Telemetría de trazas y métricas integrada con **OpenTelemetry** y estructuración de logs estructurados en JSON vía `slog`.
*   **Gestión Inteligente de Cierre de Posiciones**: Deducción dinámica del sentido (`BUY`/`SELL`) y modo (`Hedge` vs `One-way`) para el cierre exacto de posiciones en contratos de Futuros.

---

## 🛠️ Herramientas Expuestas (MCP Tools)

El servidor expone herramientas clasificadas en las siguientes áreas de trading:

| Categoría | Nombre de la Herramienta | Descripción |
| :--- | :--- | :--- |
| **Spot Trading** | `create_spot_order` | Envía órdenes Spot (MARKET, LIMIT, LIMIT_MAKER). |
| **Order Management** | `cancel_order` <br> `get_open_orders` <br> `get_order_status` <br> `get_my_trades` <br> `cancel_all_orders` | Consulta y cancelación unitaria o masiva de órdenes, e historial de fills (`get_my_trades`), en Spot o Futures vía el parámetro `market`. |
| **Risk Control** | `create_stop_loss_order` <br> `create_take_profit_order` <br> `create_stop_limit_order` <br> `create_trailing_stop_order` <br> `create_oco_order` | Estructuras avanzadas de control de pérdidas y tomas de ganancias con validaciones integradas. |
| **Market Data** | `get_ticker` <br> `get_order_book` <br> `get_klines` <br> `get_funding_rate` | Datos de mercado en tiempo real, velas japonesas y tasa de financiación de Futuros. |
| **Futures** | `create_contract_order` <br> `close_position` <br> `get_futures_positions` | Órdenes de futuros (MARKET, LIMIT, STOP_MARKET, TAKE_PROFIT_MARKET con `stopPrice`, `reduceOnly` y `closePosition`), posiciones abiertas (filtrables por `symbol`) y cierre inteligente de posiciones. |
| **Options** | `create_option_order` <br> `get_option_chain` <br> `get_option_positions` <br> `get_option_info` | Consulta y operativa de opciones europeas de Binance (solo en producción). |
| **Account** | `get_balance` <br> `get_futures_balance` <br> `get_positions` <br> `set_leverage` <br> `set_margin_mode` <br> `transfer_funds` | Administración de balances Spot y del wallet de Futuros (USDⓈ-M), apalancamiento, tipo de margen (CROSSED/ISOLATED) y transferencias universales internas. |
| **System** | `get_server_info` | Información del servidor de Binance y entorno actual. |

---

## ⚙️ Configuración (Variables de Entorno)

El servidor se configura a través de las siguientes variables de entorno:

| Variable | Tipo | Obligatorio | Por Defecto | Descripción |
| :--- | :--- | :--- | :--- | :--- |
| `BINANCE_API_KEY` | String | **Sí** | - | Clave de API de Binance para Spot y Opciones. |
| `BINANCE_SECRET_KEY` | String | **Sí** | - | Firma secreta de API de Binance correspondientes a Spot y Opciones. |
| `BINANCE_TESTNET` | Boolean | No | `false` | Activa la red de pruebas (Testnet) para Spot y Futuros. |
| `BINANCE_FUTURES_API_KEY` | String | No | *(Copia `BINANCE_API_KEY`)* | Clave de API específica para Futuros (requerido en Testnet). |
| `BINANCE_FUTURES_SECRET_KEY`| String | No | *(Copia `BINANCE_SECRET_KEY`)* | Secreto de API específico para Futuros (requerido en Testnet). |
| `LOG_FILE` | String | No | *(Path en Cache Dir / Temp)*| Ubicación física del archivo de logs y telemetría (.log). |
| `TIMEOUT_SECONDS` | Integer | No | `30` | Límite máximo de espera en segundos para solicitudes HTTP. |

---

## 🚀 Cómo Empezar

### Requisitos Previos
*   **Go** (versión 1.25+ recomendada)
*   Una cuenta de Binance con claves API habilitadas (Spot y/o Futuros).

### 1. Clonar y Compilar
```bash
git clone https://github.com/dmotta/binance-mcp-go.git
cd binance-mcp-go
go build -o binance-mcp
```

### 2. Ejecutar de forma local (Pruebas manuales)
Establece las variables requeridas y ejecuta el binario:
```bash
export BINANCE_API_KEY="tu_api_key_aqui"
export BINANCE_SECRET_KEY="tu_secret_key_aqui"
export BINANCE_TESTNET="true"

./binance-mcp
```
*Nota: El servidor arrancará a través del protocolo StdIO y estará a la escucha de peticiones JSON-RPC de un cliente MCP.*

### 3. Ejecutar las Pruebas Unitarias
El proyecto cuenta con un suite completo de pruebas unitarias que simulan la red y los puertos:
```bash
go test ./... -v
```

---

## ⚙️ Integración con Clientes MCP

### Configuración en Claude Desktop

Agrega la configuración del servidor en tu archivo `claude_desktop_config.json` (usualmente en `~/Library/Application Support/Claude/claude_desktop_config.json` en macOS):

```json
{
  "mcpServers": {
    "binance-mcp": {
      "command": "/Users/tu_usuario/Documents/ws-github-dmotta/binance-mcp-go/binance-mcp",
      "env": {
        "BINANCE_API_KEY": "V8HkAb9FBVHje...",
        "BINANCE_SECRET_KEY": "m3kAfAbzNska...",
        "BINANCE_TESTNET": "true",
        "BINANCE_FUTURES_API_KEY": "7FQ855xaBzJE...",
        "BINANCE_FUTURES_SECRET_KEY": "7FnKaaZlTneY..."
      }
    }
  }
}
```

Una vez configurado, reinicia Claude Desktop. Verás el icono del martillo indicando que las herramientas de Binance están disponibles para tu asistente.