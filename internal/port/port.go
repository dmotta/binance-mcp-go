// Package port defines the primary ports of the hexagonal architecture.
// All tool handlers depend on these interfaces, never on concrete Binance types.
package port

import "context"

// --- Spot Trading ---

type CreateSpotOrderParams struct {
	Symbol   string
	Side     string
	Type     string
	Quantity string
	Price    string // optional; empty for MARKET
}

type OrderResult struct {
	OrderID int64  `json:"orderId"`
	Symbol  string `json:"symbol"`
	Status  string `json:"status"`
}

// --- Order Management ---

type CancelOrderParams struct {
	Symbol  string
	OrderID int64
	Market  string // "spot" (default) or "futures"
}

type GetOpenOrdersParams struct {
	Symbol string // optional
	Market string // "spot" (default) or "futures"
}

type GetOrderStatusParams struct {
	Symbol  string
	OrderID int64
	Market  string // "spot" (default) or "futures"
}

type GetMyTradesParams struct {
	Symbol string
	Limit  int
	Market string // "spot" (default) or "futures"
}

type CancelAllOrdersParams struct {
	Symbol string
	Market string // "spot" (default) or "futures"
}

type Order struct {
	OrderID int64  `json:"orderId"`
	Symbol  string `json:"symbol"`
	Status  string `json:"status"`
	Side    string `json:"side"`
	Type    string `json:"type"`
	Price   string `json:"price"`
	Qty     string `json:"origQty"`
}

type Trade struct {
	ID          int64  `json:"id"`
	Symbol      string `json:"symbol"`
	Price       string `json:"price"`
	Qty         string `json:"qty"`
	IsBuyer     bool   `json:"isBuyer"`
	Time        int64  `json:"time"`
	Side        string `json:"side,omitempty"`        // futures only
	RealizedPnl string `json:"realizedPnl,omitempty"` // futures only
}

// --- Risk Control ---

type StopOrderParams struct {
	Symbol    string
	Side      string
	Quantity  string
	StopPrice string
}

type StopLimitOrderParams struct {
	Symbol     string
	Side       string
	Quantity   string
	StopPrice  string
	LimitPrice string
}

type TrailingStopOrderParams struct {
	Symbol       string
	Side         string
	Quantity     string
	CallbackRate float64 // percentage, e.g. 1.5 for 1.5%
}

type OCOOrderParams struct {
	Symbol         string
	Side           string
	Quantity       string
	Price          string
	StopPrice      string
	StopLimitPrice string
}

type OCOResult struct {
	OrderListID int64    `json:"orderListId"`
	Symbol      string   `json:"symbol"`
	Orders      []Order  `json:"orders"`
}

// --- Market Data ---

type Ticker struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	LastPrice          string `json:"lastPrice"`
	Volume             string `json:"volume"`
}

type OrderBook struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

type Kline struct {
	OpenTime  int64  `json:"openTime"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
	CloseTime int64  `json:"closeTime"`
}

type FundingRate struct {
	Symbol      string `json:"symbol"`
	FundingRate string `json:"fundingRate"`
	FundingTime int64  `json:"fundingTime"`
}

// --- Options ---

type OptionOrderParams struct {
	Symbol   string
	Side     string
	Quantity string
	Price    string
}

type OptionOrder struct {
	OrderID  string `json:"orderId"`
	Symbol   string `json:"symbol"`
	Status   string `json:"status"`
	Price    string `json:"price"`
	Quantity string `json:"quantity"`
}

type OptionSymbol struct {
	Symbol     string `json:"symbol"`
	Underlying string `json:"underlying"`
	StrikePrice string `json:"strikePrice"`
	ExpiryDate  int64  `json:"expiryDate"`
}

type OptionPosition struct {
	Symbol        string `json:"symbol"`
	PositionAmt   string `json:"positionAmt"`
	AvgPrice      string `json:"avgPrice"`
	UnrealizedPnL string `json:"unrealizedPnl"`
}

// --- Futures ---

type ContractOrderParams struct {
	Symbol        string
	Side          string
	Type          string
	Quantity      string // empty allowed only with ClosePosition
	Price         string
	StopPrice     string // required for STOP_MARKET / TAKE_PROFIT_MARKET
	ReduceOnly    bool
	ClosePosition bool // STOP_MARKET / TAKE_PROFIT_MARKET only; excludes Quantity and ReduceOnly
}

type FuturesPosition struct {
	Symbol           string `json:"symbol"`
	PositionAmt      string `json:"positionAmt"`
	EntryPrice       string `json:"entryPrice"`
	UnRealizedProfit string `json:"unrealizedPnl"`
	LiquidationPrice string `json:"liquidationPrice"`
	Leverage         string `json:"leverage"`
	PositionSide     string `json:"positionSide"`
}

// --- Account ---

type Balance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

type FuturesBalance struct {
	Asset            string `json:"asset"`
	Balance          string `json:"balance"`
	CrossUnPnl       string `json:"crossUnPnl"`
	AvailableBalance string `json:"availableBalance"`
}

type SetLeverageParams struct {
	Symbol   string
	Leverage int
}

type SetMarginModeParams struct {
	Symbol     string
	MarginMode string
}

type TransferFundsParams struct {
	Asset       string
	Amount      string
	FromAccount string
	ToAccount   string
}

// --- System ---

type ServerInfo struct {
	ServerTime  int64  `json:"serverTime"`
	Version     string `json:"version"`
	Environment string `json:"environment"`
}

// BinancePort is the primary port: everything a tool handler needs.
type BinancePort interface {
	// Spot Trading
	CreateSpotOrder(ctx context.Context, p CreateSpotOrderParams) (*OrderResult, error)

	// Order Management
	CancelOrder(ctx context.Context, p CancelOrderParams) (*Order, error)
	GetOpenOrders(ctx context.Context, p GetOpenOrdersParams) ([]Order, error)
	GetOrderStatus(ctx context.Context, p GetOrderStatusParams) (*Order, error)
	GetMyTrades(ctx context.Context, p GetMyTradesParams) ([]Trade, error)
	CancelAllOrders(ctx context.Context, p CancelAllOrdersParams) ([]Order, error)

	// Risk Control
	CreateStopLossOrder(ctx context.Context, p StopOrderParams) (*OrderResult, error)
	CreateTakeProfitOrder(ctx context.Context, p StopOrderParams) (*OrderResult, error)
	CreateStopLimitOrder(ctx context.Context, p StopLimitOrderParams) (*OrderResult, error)
	CreateTrailingStopOrder(ctx context.Context, p TrailingStopOrderParams) (*OrderResult, error)
	CreateOCOOrder(ctx context.Context, p OCOOrderParams) (*OCOResult, error)

	// Market Data
	GetTicker(ctx context.Context, symbol string) (*Ticker, error)
	GetOrderBook(ctx context.Context, symbol string, limit int) (*OrderBook, error)
	GetKlines(ctx context.Context, symbol, interval string, limit int) ([]Kline, error)
	GetFundingRate(ctx context.Context, symbol string) (*FundingRate, error)

	// Options
	CreateOptionOrder(ctx context.Context, p OptionOrderParams) (*OptionOrder, error)
	GetOptionChain(ctx context.Context, underlying string) ([]OptionSymbol, error)
	GetOptionPositions(ctx context.Context) ([]OptionPosition, error)
	GetOptionInfo(ctx context.Context, symbol string) (*OptionSymbol, error)

	// Futures
	CreateContractOrder(ctx context.Context, p ContractOrderParams) (*OrderResult, error)
	ClosePosition(ctx context.Context, symbol string) (*OrderResult, error)
	GetFuturesPositions(ctx context.Context, symbol string) ([]FuturesPosition, error)

	// Account
	GetBalance(ctx context.Context, asset string) ([]Balance, error)
	GetFuturesBalance(ctx context.Context, asset string) ([]FuturesBalance, error)
	GetPositions(ctx context.Context) ([]FuturesPosition, error)
	SetLeverage(ctx context.Context, p SetLeverageParams) error
	SetMarginMode(ctx context.Context, p SetMarginModeParams) error
	TransferFunds(ctx context.Context, p TransferFundsParams) error

	// System
	GetServerInfo(ctx context.Context) (*ServerInfo, error)
}
