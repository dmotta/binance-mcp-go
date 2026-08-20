package tools

import (
	"context"
	"errors"

	"binance-mcp-go/internal/port"
)

// mockPort implements port.BinancePort for testing.
// Each field holds the value to return; set ErrX to simulate errors.
type mockPort struct {
	// return values
	spotOrderResult    *port.OrderResult
	order              *port.Order
	orders             []port.Order
	trades             []port.Trade
	ocoResult          *port.OCOResult
	ticker             *port.Ticker
	orderBook          *port.OrderBook
	klines             []port.Kline
	fundingRate        *port.FundingRate
	optionOrder        *port.OptionOrder
	optionSymbols      []port.OptionSymbol
	optionPositions    []port.OptionPosition
	optionSymbol       *port.OptionSymbol
	futuresPositions   []port.FuturesPosition
	algoOrders         []port.AlgoOrder
	algoCancelResult   *port.AlgoCancelResult
	balances           []port.Balance
	futuresBalances    []port.FuturesBalance
	serverInfo         *port.ServerInfo

	// errors
	errSpotOrder        error
	errCancelOrder      error
	errOpenOrders       error
	errOrderStatus      error
	errMyTrades         error
	errCancelAll        error
	errStopLoss         error
	errTakeProfit       error
	errStopLimit        error
	errTrailingStop     error
	errOCO              error
	errTicker           error
	errOrderBook        error
	errKlines           error
	errFundingRate      error
	errOptionOrder      error
	errOptionChain      error
	errOptionPositions  error
	errOptionInfo       error
	errContractOrder    error
	errClosePosition    error
	errFuturesPositions error
	errAlgoOrders       error
	errCancelAlgoOrder  error
	errBalance          error
	errFuturesBalance   error
	errPositions        error
	errSetLeverage      error
	errSetMarginMode    error
	errTransferFunds    error
	errServerInfo       error

	// capture last call params for assertion
	lastCreateSpotOrder     *port.CreateSpotOrderParams
	lastCancelOrder         *port.CancelOrderParams
	lastGetOpenOrders       *port.GetOpenOrdersParams
	lastGetOrderStatus      *port.GetOrderStatusParams
	lastGetMyTrades         *port.GetMyTradesParams
	lastCancelAllOrders     *port.CancelAllOrdersParams
	lastTrailingStop        *port.TrailingStopOrderParams
	lastContractOrder       *port.ContractOrderParams
	lastClosePositionSymbol string
	lastFuturesPosSymbol    string
	lastAlgoOrdersSymbol    string
	lastCancelAlgoID        int64
	lastFuturesBalAsset     string
	lastSetLeverage         *port.SetLeverageParams
	lastSetMarginMode       *port.SetMarginModeParams
	lastTransferFunds       *port.TransferFundsParams
}

func (m *mockPort) CreateSpotOrder(ctx context.Context, p port.CreateSpotOrderParams) (*port.OrderResult, error) {
	m.lastCreateSpotOrder = &p
	return m.spotOrderResult, m.errSpotOrder
}
func (m *mockPort) CancelOrder(ctx context.Context, p port.CancelOrderParams) (*port.Order, error) {
	m.lastCancelOrder = &p
	return m.order, m.errCancelOrder
}
func (m *mockPort) GetOpenOrders(ctx context.Context, p port.GetOpenOrdersParams) ([]port.Order, error) {
	m.lastGetOpenOrders = &p
	return m.orders, m.errOpenOrders
}
func (m *mockPort) GetOrderStatus(ctx context.Context, p port.GetOrderStatusParams) (*port.Order, error) {
	m.lastGetOrderStatus = &p
	return m.order, m.errOrderStatus
}
func (m *mockPort) GetMyTrades(ctx context.Context, p port.GetMyTradesParams) ([]port.Trade, error) {
	m.lastGetMyTrades = &p
	return m.trades, m.errMyTrades
}
func (m *mockPort) CancelAllOrders(ctx context.Context, p port.CancelAllOrdersParams) ([]port.Order, error) {
	m.lastCancelAllOrders = &p
	return m.orders, m.errCancelAll
}
func (m *mockPort) CreateStopLossOrder(ctx context.Context, p port.StopOrderParams) (*port.OrderResult, error) {
	return m.spotOrderResult, m.errStopLoss
}
func (m *mockPort) CreateTakeProfitOrder(ctx context.Context, p port.StopOrderParams) (*port.OrderResult, error) {
	return m.spotOrderResult, m.errTakeProfit
}
func (m *mockPort) CreateStopLimitOrder(ctx context.Context, p port.StopLimitOrderParams) (*port.OrderResult, error) {
	return m.spotOrderResult, m.errStopLimit
}
func (m *mockPort) CreateTrailingStopOrder(ctx context.Context, p port.TrailingStopOrderParams) (*port.OrderResult, error) {
	m.lastTrailingStop = &p
	return m.spotOrderResult, m.errTrailingStop
}
func (m *mockPort) CreateOCOOrder(ctx context.Context, p port.OCOOrderParams) (*port.OCOResult, error) {
	return m.ocoResult, m.errOCO
}
func (m *mockPort) GetTicker(ctx context.Context, symbol string) (*port.Ticker, error) {
	return m.ticker, m.errTicker
}
func (m *mockPort) GetOrderBook(ctx context.Context, symbol string, limit int) (*port.OrderBook, error) {
	return m.orderBook, m.errOrderBook
}
func (m *mockPort) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]port.Kline, error) {
	return m.klines, m.errKlines
}
func (m *mockPort) GetFundingRate(ctx context.Context, symbol string) (*port.FundingRate, error) {
	return m.fundingRate, m.errFundingRate
}
func (m *mockPort) CreateOptionOrder(ctx context.Context, p port.OptionOrderParams) (*port.OptionOrder, error) {
	return m.optionOrder, m.errOptionOrder
}
func (m *mockPort) GetOptionChain(ctx context.Context, underlying string) ([]port.OptionSymbol, error) {
	return m.optionSymbols, m.errOptionChain
}
func (m *mockPort) GetOptionPositions(ctx context.Context) ([]port.OptionPosition, error) {
	return m.optionPositions, m.errOptionPositions
}
func (m *mockPort) GetOptionInfo(ctx context.Context, symbol string) (*port.OptionSymbol, error) {
	return m.optionSymbol, m.errOptionInfo
}
func (m *mockPort) CreateContractOrder(ctx context.Context, p port.ContractOrderParams) (*port.OrderResult, error) {
	m.lastContractOrder = &p
	return m.spotOrderResult, m.errContractOrder
}
func (m *mockPort) ClosePosition(ctx context.Context, symbol string) (*port.OrderResult, error) {
	m.lastClosePositionSymbol = symbol
	return m.spotOrderResult, m.errClosePosition
}
func (m *mockPort) GetFuturesPositions(ctx context.Context, symbol string) ([]port.FuturesPosition, error) {
	m.lastFuturesPosSymbol = symbol
	return m.futuresPositions, m.errFuturesPositions
}
func (m *mockPort) GetOpenAlgoOrders(ctx context.Context, symbol string) ([]port.AlgoOrder, error) {
	m.lastAlgoOrdersSymbol = symbol
	return m.algoOrders, m.errAlgoOrders
}
func (m *mockPort) CancelAlgoOrder(ctx context.Context, algoID int64) (*port.AlgoCancelResult, error) {
	m.lastCancelAlgoID = algoID
	return m.algoCancelResult, m.errCancelAlgoOrder
}
func (m *mockPort) GetBalance(ctx context.Context, asset string) ([]port.Balance, error) {
	return m.balances, m.errBalance
}
func (m *mockPort) GetFuturesBalance(ctx context.Context, asset string) ([]port.FuturesBalance, error) {
	m.lastFuturesBalAsset = asset
	return m.futuresBalances, m.errFuturesBalance
}
func (m *mockPort) GetPositions(ctx context.Context) ([]port.FuturesPosition, error) {
	return m.futuresPositions, m.errPositions
}
func (m *mockPort) SetLeverage(ctx context.Context, p port.SetLeverageParams) error {
	m.lastSetLeverage = &p
	return m.errSetLeverage
}
func (m *mockPort) SetMarginMode(ctx context.Context, p port.SetMarginModeParams) error {
	m.lastSetMarginMode = &p
	return m.errSetMarginMode
}
func (m *mockPort) TransferFunds(ctx context.Context, p port.TransferFundsParams) error {
	m.lastTransferFunds = &p
	return m.errTransferFunds
}
func (m *mockPort) GetServerInfo(ctx context.Context) (*port.ServerInfo, error) {
	return m.serverInfo, m.errServerInfo
}

// compile-time check
var _ port.BinancePort = (*mockPort)(nil)

// sentinelErr is a generic test error.
var sentinelErr = errors.New("binance error")
