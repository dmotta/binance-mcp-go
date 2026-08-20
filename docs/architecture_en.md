# Architecture & Go Code Explanation

This document provides a comprehensive architectural breakdown and detailed explanation of each module within **binance-mcp-go**, serving as a reference guide to understand the system design and the internal mechanics of the Go codebase.

---

## 🏛️ Architectural Design: Hexagonal Architecture

The project is structured around the principles of **Hexagonal Architecture (also known as Ports & Adapters)**. This pattern divides the application into three distinct layers:

1.  **Core / Ports (`internal/port`)**: Declares abstract interfaces representing what operations the system can perform. The rest of the application (e.g. MCP tool handlers) depends strictly on these interfaces, remaining completely decoupled from third-party SDKs or direct HTTP clients.
2.  **Adapters (`internal/adapter`)**: Contains concrete implementations of the ports. This is where the external SDK `github.com/adshao/go-binance/v2` is integrated to interact with Binance's API endpoints.
3.  **Presentation / Driving Layer (`internal/tools` & `main.go`)**: Receives JSON-RPC requests from the MCP client, unpacks parameters, calls the appropriate port method, and formats the output.

### Benefits of this Pattern
*   **Testability**: MCP tool logic can be fully tested without hitting actual network endpoints by injecting a mock implementing `BinancePort` (as demonstrated in `internal/tools/mock_port_test.go`).
*   **Maintainability**: If the Binance SDK were to change or be replaced with raw HTTP calls, only the adapter package needs modification; the MCP tools and general server orchestration remain untouched.

---

## 📊 Architecture Diagrams (C4 Model)

To illustrate the data flows, container divisions, and internal Go package components, the 4 levels of the **C4 Model** are presented below.

### 🛠️ Level 1: System Context
Shows how a user (or an autonomous AI agent) interacts with Claude Desktop, the Binance MCP server, and the external Binance API.

```mermaid
graph TD
    User["👤 User / AI Agent<br>(Person)"] -->|"Uses natural language commands or UI"| Client["💻 Claude Desktop / MCP Client<br>(Software System)"]
    Client -->|"Starts and consumes tools via Stdio"| MCPServer["🚀 Binance MCP Server (Go)<br>(Software System - This project)"]
    MCPServer -->|"HTTP REST requests (signed and public)"| BinanceAPI["🌐 Binance API Platform<br>(External Software System)"]

    style User fill:#08427b,stroke:#052e56,color:#fff
    style Client fill:#1168bd,stroke:#0b4884,color:#fff
    style MCPServer fill:#1168bd,stroke:#0b4884,color:#fff
    style BinanceAPI fill:#999999,stroke:#666666,color:#fff
```

---

### 📦 Level 2: Containers
Illustrates the physical boundaries of the deployment, showing local configuration and logs on the user's machine relative to the remote servers.

```mermaid
graph TB
    subgraph LocalMachine [User's Local Machine]
        Claude["💻 Claude Desktop / MCP Client<br>(Electron App / UI)<br><br>Graphical chat interface that executes local tool subprocesses."]

        subgraph GoAppContainer [Container: Go Application]
            MCPServerGo["⚙️ Binance MCP Executable<br>(Go Binary CLI / Stdio)<br><br>Listens to JSON-RPC commands over standard input/output and coordinates calls."]
        end

        EnvVars["📄 Environment Variables<br>(Configuration)<br><br>BINANCE_API_KEY, BINANCE_SECRET_KEY, BINANCE_TESTNET, etc."]
        LogFile["📝 Log File (.log)<br><br>Persistently stores slog JSON logs, traces, and metrics from the server."]

        Claude -->|"JSON-RPC commands via Stdio / IPC"| MCPServerGo
        EnvVars -->|"Read at process startup"| MCPServerGo
        MCPServerGo -->|"Writes logs and telemetry (OTel)"| LogFile
    end

    BinanceAPI["🌐 Binance API Endpoints<br>(REST API)<br><br>Global external Binance REST endpoints for Spot, Futures, and Options."]
    MCPServerGo -->|"Secure HTTP requests over TLS (Port 443)"| BinanceAPI

    style Claude fill:#1168bd,stroke:#0b4884,color:#fff
    style MCPServerGo fill:#1168bd,stroke:#0b4884,color:#fff
    style EnvVars fill:#2b822b,stroke:#1e5c1e,color:#fff
    style LogFile fill:#2b822b,stroke:#1e5c1e,color:#fff
    style BinanceAPI fill:#999999,stroke:#666666,color:#fff
```

---

### 🧩 Level 3: Components
Deconstructs the internal packages inside the `Binance MCP` Go executable and details how dependencies are wired between the MCP layer, the hexagonal ports, and the resilient transport layer.

```mermaid
graph TD
    Stdio[Stdio / IPC] <-->|JSON-RPC| MCPServer["mcp-go Server Component<br>(Library)"]
    MCPServer -->|"Invokes registered tool handlers"| Tools["Tool Handler<br>(internal/tools)<br><br>Defines JSON schemas, validates parameter types, and maps data."]
    
    Tools -->|"Calls port abstraction"| Port["BinancePort Interface<br>(internal/port)<br><br>Hexagonal domain port. Declares all trading contracts."]
    
    Adapter["BinanceAdapter<br>(internal/adapter)<br><br>Domain adapter. Translates port calls to the external SDK library."] -.->|Implements| Port
    
    Adapter -->|"Calls SDK client"| GoBinanceSDK["go-binance SDK Client<br>(adshao/go-binance/v2)"]
    
    GoBinanceSDK -->|"Performs REST requests using"| HttpClient["Custom http.Client"]
    
    subgraph HTTPMiddlewareChain [Transport Resilience Layer: internal/httpmw]
        HttpClient -->|"1. OTel Transport"| OtelTransport["otelTransport<br>(Network traces and metrics)"]
        OtelTransport -->|"2. Circuit Breaker"| CBTransport["circuitBreakerTransport<br>(sony/gobreaker, failures>=5)"]
        CBTransport -->|"3. Retry Transport"| RetryTransport["retryTransport<br>(Safe exponential retries)"]
        RetryTransport -->|"4. Rate Limit Transport"| RLTransport["rateLimitTransport<br>(Binance request weight limits control)"]
    end
    
    RLTransport -->|"Physical HTTP call"| Net[External Network / Binance API]

    Config["Config Component<br>(internal/config)<br><br>Loads environment variables and injects them."] -.->|"Configures credentials"| Adapter
    Observability["Observability Component<br>(internal/observability)<br><br>Manages slog and OpenTelemetry."] -.->|"Logger and initialization"| MCPServer
    Observability -.->|"Configures writer and exporter"| OtelTransport

    style MCPServer fill:#1168bd,stroke:#0b4884,color:#fff
    style Tools fill:#1168bd,stroke:#0b4884,color:#fff
    style Port fill:#0288d1,stroke:#01579b,color:#fff
    style Adapter fill:#2e7d32,stroke:#1b5e20,color:#fff
    style GoBinanceSDK fill:#ef6c00,stroke:#e65100,color:#fff
    style OtelTransport fill:#8e24aa,stroke:#4a148c,color:#fff
    style CBTransport fill:#8e24aa,stroke:#4a148c,color:#fff
    style RetryTransport fill:#8e24aa,stroke:#4a148c,color:#fff
    style RLTransport fill:#8e24aa,stroke:#4a148c,color:#fff
    style Config fill:#546e7a,stroke:#263238,color:#fff
    style Observability fill:#546e7a,stroke:#263238,color:#fff
```

---

### 💻 Level 4: Code
Shows the structural relationship between Go types, detailing the implementation of `BinancePort` and the chain of custom `http.RoundTripper` decorator implementations.

```mermaid
classDiagram
    class http_RoundTripper {
        <<interface>>
        +RoundTrip(req *http.Request) (*http.Response, error)
    }

    class BinancePort {
        <<interface>>
        +CreateSpotOrder(ctx, p CreateSpotOrderParams) (*OrderResult, error)
        +CancelOrder(ctx, p CancelOrderParams) (*Order, error)
        +GetOpenOrders(ctx, p GetOpenOrdersParams) ([]Order, error)
        +GetOrderStatus(ctx, p GetOrderStatusParams) (*Order, error)
        +ClosePosition(ctx, symbol string) (*OrderResult, error)
        +GetBalance(ctx, asset string) ([]Balance, error)
        +SetLeverage(ctx, p SetLeverageParams) error
    }

    class BinanceAdapter {
        -spot: *binance.Client
        -futures: *futures.Client
        -opts: *options.Client
        -env: string
        +New(spot, fut, opts, env) *BinanceAdapter
        +CreateSpotOrder(...)
        +ClosePosition(...)
    }

    class otelTransport {
        -next: http.RoundTripper
        -tracer: trace.Tracer
        -counter: metric.Int64Counter
        +RoundTrip(...)
    }

    class circuitBreakerTransport {
        -next: http.RoundTripper
        -cb: *gobreaker.CircuitBreaker
        +RoundTrip(...)
    }

    class retryTransport {
        -next: http.RoundTripper
        +RoundTrip(...)
    }

    class rateLimitTransport {
        -next: http.RoundTripper
        -usedWeight: atomic.Int64
        -sleep: func(time.Duration)
        +RoundTrip(...)
    }

    BinanceAdapter ..|> BinancePort : implements
    otelTransport ..|> http_RoundTripper : implements
    circuitBreakerTransport ..|> http_RoundTripper : implements
    retryTransport ..|> http_RoundTripper : implements
    rateLimitTransport ..|> http_RoundTripper : implements

    otelTransport --> http_RoundTripper : decorates (next)
    circuitBreakerTransport --> http_RoundTripper : decorates (next)
    retryTransport --> http_RoundTripper : decorates (next)
    rateLimitTransport --> http_RoundTripper : decorates (next)
```

---

## 📂 Detailed Module Breakdown

### 1. Entry Point (`main.go`)
[main.go](../main.go) orchestrates the application startup:
*   **Context Setup**: Sets up signal listening (`SIGTERM`, `os.Interrupt`) using a cancelable context to ensure clean shutdowns.
*   **Config Loading**: Loads variables from the environment (`config.Load()`).
*   **Observability Startup**: Initializes slog logging and OpenTelemetry.
*   **Resilient HTTP Transport Chain**: Instantiates custom middlewares for OTel, Circuit Breakers, Retries, and Rate Limits.
*   **Binance SDK Client Setup**: Instantiates separate SDK clients for Spot, Futures, and Options, injecting the resilient HTTP client.
*   **Dependency Injection**: Instantiates the adapter (`adapter.New(...)`) and registers all JSON-RPC tools (`tools.RegisterAll(...)`).
*   **Server Run**: Starts the MCP server on Stdio.

---

### 2. Configuration (`internal/config`)
Defined in [config.go](../internal/config/config.go), this package reads configuration parameters from environment variables:
*   `BINANCE_API_KEY` and `BINANCE_SECRET_KEY`: Required credentials.
*   `BINANCE_TESTNET`: Specifies whether the server directs requests to the Binance Testnet domains.
*   `TIMEOUT_SECONDS`: Maximum HTTP request wait time (defaults to 30 seconds).
*   `BINANCE_FUTURES_API_KEY` and `BINANCE_FUTURES_SECRET_KEY`: In Testnet, Futures API keys are distinct from Spot. These fall back to the Spot keys if not provided.
*   `defaultLogPath()`: Places the logs directory inside the operating system's local user cache (e.g. `Library/Caches/binance-mcp/` on macOS).

---

### 3. Observability (`internal/observability`)
Defined in [observability.go](../internal/observability/observability.go), this module configures local telemetry:
*   **JSON logging**: Creates a structured log engine using `slog` outputting JSON directly to the log file.
*   **OpenTelemetry Tracing**: Sets up a tracer writing spans to the log file in JSON.
*   **OpenTelemetry Metrics**: Registers periodic readers to emit network metrics.
*   **Shutdown Handler**: Flushes OpenTelemetry buffers to prevent losing traces during a shutdown.

---

### 4. HTTP Transport Middlewares (`internal/httpmw`)
Middlewares intercept outgoing HTTP calls by wrapping `http.RoundTripper` in a decorator pattern. The chain is constructed in [chain.go](../internal/httpmw/chain.go) as follows:

#### A. Telemetry (`otelhttp.go`)
[otelhttp.go](../internal/httpmw/otelhttp.go) intercepts outgoing requests to record HTTP methods, URL paths, and status codes. It increments the `http.client.requests` counter and tags errors on failed network requests.

#### B. Circuit Breaker (`circuitbreaker.go`)
Uses `github.com/sony/gobreaker/v2` in [circuitbreaker.go](../internal/httpmw/circuitbreaker.go). If the Binance API returns 5xx status codes (or transport failures) 5 consecutive times, the breaker trips to **Open**, preventing additional network calls locally during a cool-down period.

#### C. Exponential Retries (`retry.go`)
Defined in [retry.go](../internal/httpmw/retry.go):
*   Attempts requests up to **3 times** with exponential backoff and randomized jitter to prevent thundering herd conditions.
*   **Idempotency Safety**: Verifies if the request body is rewindable (`req.GetBody != nil`). If a request body cannot be rewound (e.g., streaming payloads), the retry is aborted. This prevents duplicate executions on orders, protecting the wallet against unintended duplicate trades.

#### D. Rate Limiting (`ratelimit.go`)
Defined in [ratelimit.go](../internal/httpmw/ratelimit.go):
*   Binance enforces a request weight limit of 1200 per minute. This middleware reads the `X-Mbx-Used-Weight-1M` header from responses to track used weight.
*   If used weight crosses the warning threshold (`weightThreshold = 1100`), the middleware blocks new requests until the next minute boundary to avoid receiving IP bans (HTTP 418/429).
*   If an HTTP 429/418 is received, it parses the `Retry-After` header and sleeps for the requested duration.

---

### 5. Hexagonal Port (`internal/port`)
Declared in [port.go](../internal/port/port.go), `BinancePort` is the interface isolating core domain models (`Order`, `Balance`, `Ticker`, `Kline`) from the specific API implementation details.

---

### 6. Binance SDK Adapter (`internal/adapter`)
Written in [adapter.go](../internal/adapter/adapter.go), `BinanceAdapter` implements `BinancePort` by calling the `go-binance` library:
*   **Options Guard (`optionsGuard`)**: Since Binance does not offer a public Options testnet, this helper returns an error if options tools are triggered in testnet mode.
*   **Position Closing Logic (`ClosePosition`)**: Close actions require calculating exact opposite sides and checking position settings:
    1.  Queries active positions via `NewGetPositionRiskService()`.
    2.  Resolves the direction from the signed position size (`PositionAmt`). A positive number represents a Long, while a negative number indicates a Short.
    3.  Determines the closing order side (e.g., if Long, the closing order is a `SELL` market order).
    4.  Inspects if the account is in **Hedge Mode** (requiring explicit LONG/SHORT position side flags) or **One-Way Mode** (requiring the `ReduceOnly` flag to ensure the position is reduced without flipping into a counter-position).

---

### 7. MCP Tools Layer (`internal/tools`)
The `internal/tools` package registers JSON-RPC tool endpoints with the MCP server (e.g., [spot_trading.go](../internal/tools/spot_trading.go), [risk_control.go](../internal/tools/risk_control.go), [account.go](../internal/tools/account.go)).

*   **Schema Registration**: Calls `toolWithRawSchema` to register raw JSON schemas.
*   **Type Conversions**: Tool arguments are parsed from JSON values. Quantities and prices are parsed, validated, and formatted into strings (`%g`) to feed the SDK.
*   **Trailing Stop Handling**: The `create_trailing_stop_order` tool accepts `callbackRate` in percent (e.g. `1.5` for 1.5%), validating it is in the range `0.1-20`. The adapter automatically converts this to basis points (BIPS) (e.g. `150`) as required by the Binance API.
