// Package adapter implements port.BinancePort using the go-binance SDK.
package adapter

import (
	"context"
	"fmt"
	"math"
	"strconv"

	binance "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/adshao/go-binance/v2/options"

	"binance-mcp-go/internal/port"
)

// BinanceAdapter implements port.BinancePort.
type BinanceAdapter struct {
	spot    *binance.Client
	futures *futures.Client
	opts    *options.Client
	env     string
}

func New(spot *binance.Client, fut *futures.Client, opts *options.Client, env string) *BinanceAdapter {
	return &BinanceAdapter{spot: spot, futures: fut, opts: opts, env: env}
}

// ─── Spot Trading ────────────────────────────────────────────────────────────

func (a *BinanceAdapter) CreateSpotOrder(ctx context.Context, p port.CreateSpotOrderParams) (*port.OrderResult, error) {
	svc := a.spot.NewCreateOrderService().
		Symbol(p.Symbol).
		Side(binance.SideType(p.Side)).
		Type(binance.OrderType(p.Type)).
		Quantity(p.Quantity)
	if p.Price != "" {
		svc = svc.Price(p.Price)
	}
	if p.Type == "LIMIT" || p.Type == "LIMIT_MAKER" {
		svc = svc.TimeInForce(binance.TimeInForceTypeGTC)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.OrderResult{OrderID: res.OrderID, Symbol: res.Symbol, Status: string(res.Status)}, nil
}

// ─── Order Management ────────────────────────────────────────────────────────

func (a *BinanceAdapter) CancelOrder(ctx context.Context, p port.CancelOrderParams) (*port.Order, error) {
	if p.Market == "futures" {
		return a.cancelFuturesOrder(ctx, p)
	}
	res, err := a.spot.NewCancelOrderService().
		Symbol(p.Symbol).
		OrderID(p.OrderID).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.Order{
		OrderID: res.OrderID,
		Symbol:  res.Symbol,
		Status:  string(res.Status),
		Side:    string(res.Side),
		Type:    string(res.Type),
		Price:   res.Price,
		Qty:     res.OrigQuantity,
	}, nil
}

func (a *BinanceAdapter) cancelFuturesOrder(ctx context.Context, p port.CancelOrderParams) (*port.Order, error) {
	res, err := a.futures.NewCancelOrderService().
		Symbol(p.Symbol).
		OrderID(p.OrderID).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.Order{
		OrderID: res.OrderID,
		Symbol:  res.Symbol,
		Status:  string(res.Status),
		Side:    string(res.Side),
		Type:    string(res.Type),
		Price:   res.Price,
		Qty:     res.OrigQuantity,
	}, nil
}

func (a *BinanceAdapter) GetOpenOrders(ctx context.Context, p port.GetOpenOrdersParams) ([]port.Order, error) {
	if p.Market == "futures" {
		return a.getFuturesOpenOrders(ctx, p)
	}
	svc := a.spot.NewListOpenOrdersService()
	if p.Symbol != "" {
		svc = svc.Symbol(p.Symbol)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]port.Order, len(res))
	for i, o := range res {
		out[i] = port.Order{
			OrderID: o.OrderID,
			Symbol:  o.Symbol,
			Status:  string(o.Status),
			Side:    string(o.Side),
			Type:    string(o.Type),
			Price:   o.Price,
			Qty:     o.OrigQuantity,
		}
	}
	return out, nil
}

func (a *BinanceAdapter) getFuturesOpenOrders(ctx context.Context, p port.GetOpenOrdersParams) ([]port.Order, error) {
	svc := a.futures.NewListOpenOrdersService()
	if p.Symbol != "" {
		svc = svc.Symbol(p.Symbol)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]port.Order, len(res))
	for i, o := range res {
		out[i] = port.Order{
			OrderID: o.OrderID,
			Symbol:  o.Symbol,
			Status:  string(o.Status),
			Side:    string(o.Side),
			Type:    string(o.Type),
			Price:   o.Price,
			Qty:     o.OrigQuantity,
		}
	}
	return out, nil
}

func (a *BinanceAdapter) GetOrderStatus(ctx context.Context, p port.GetOrderStatusParams) (*port.Order, error) {
	if p.Market == "futures" {
		return a.getFuturesOrderStatus(ctx, p)
	}
	res, err := a.spot.NewGetOrderService().
		Symbol(p.Symbol).
		OrderID(p.OrderID).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.Order{
		OrderID: res.OrderID,
		Symbol:  res.Symbol,
		Status:  string(res.Status),
		Side:    string(res.Side),
		Type:    string(res.Type),
		Price:   res.Price,
		Qty:     res.OrigQuantity,
	}, nil
}

func (a *BinanceAdapter) getFuturesOrderStatus(ctx context.Context, p port.GetOrderStatusParams) (*port.Order, error) {
	res, err := a.futures.NewGetOrderService().
		Symbol(p.Symbol).
		OrderID(p.OrderID).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.Order{
		OrderID: res.OrderID,
		Symbol:  res.Symbol,
		Status:  string(res.Status),
		Side:    string(res.Side),
		Type:    string(res.Type),
		Price:   res.Price,
		Qty:     res.OrigQuantity,
	}, nil
}

func (a *BinanceAdapter) GetMyTrades(ctx context.Context, p port.GetMyTradesParams) ([]port.Trade, error) {
	if p.Market == "futures" {
		return a.getFuturesMyTrades(ctx, p)
	}
	svc := a.spot.NewListTradesService().Symbol(p.Symbol)
	if p.Limit > 0 {
		svc = svc.Limit(p.Limit)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]port.Trade, len(res))
	for i, t := range res {
		out[i] = port.Trade{
			ID:      t.ID,
			Symbol:  t.Symbol,
			Price:   t.Price,
			Qty:     t.Quantity,
			IsBuyer: t.IsBuyer,
			Time:    t.Time,
		}
	}
	return out, nil
}

func (a *BinanceAdapter) getFuturesMyTrades(ctx context.Context, p port.GetMyTradesParams) ([]port.Trade, error) {
	svc := a.futures.NewListAccountTradeService().Symbol(p.Symbol)
	if p.Limit > 0 {
		svc = svc.Limit(p.Limit)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]port.Trade, len(res))
	for i, t := range res {
		out[i] = port.Trade{
			ID:          t.ID,
			Symbol:      t.Symbol,
			Price:       t.Price,
			Qty:         t.Quantity,
			IsBuyer:     t.Buyer,
			Time:        t.Time,
			Side:        string(t.Side),
			RealizedPnl: t.RealizedPnl,
		}
	}
	return out, nil
}

func (a *BinanceAdapter) CancelAllOrders(ctx context.Context, p port.CancelAllOrdersParams) ([]port.Order, error) {
	if p.Market == "futures" {
		return a.cancelAllFuturesOrders(ctx, p)
	}
	res, err := a.spot.NewCancelOpenOrdersService().Symbol(p.Symbol).Do(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]port.Order, len(res.Orders))
	for i, o := range res.Orders {
		out[i] = port.Order{
			OrderID: o.OrderID,
			Symbol:  o.Symbol,
			Status:  string(o.Status),
			Side:    string(o.Side),
			Type:    string(o.Type),
			Price:   o.Price,
			Qty:     o.OrigQuantity,
		}
	}
	return out, nil
}

func (a *BinanceAdapter) cancelAllFuturesOrders(ctx context.Context, p port.CancelAllOrdersParams) ([]port.Order, error) {
	err := a.futures.NewCancelAllOpenOrdersService().Symbol(p.Symbol).Do(ctx)
	if err != nil {
		return nil, err
	}
	// The futures API returns no per-order detail on bulk cancel.
	return []port.Order{{Symbol: p.Symbol, Status: "CANCELED"}}, nil
}

// ─── Risk Control ─────────────────────────────────────────────────────────────

func (a *BinanceAdapter) CreateStopLossOrder(ctx context.Context, p port.StopOrderParams) (*port.OrderResult, error) {
	res, err := a.spot.NewCreateOrderService().
		Symbol(p.Symbol).
		Side(binance.SideType(p.Side)).
		Type(binance.OrderTypeStopLoss).
		Quantity(p.Quantity).
		StopPrice(p.StopPrice).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.OrderResult{OrderID: res.OrderID, Symbol: res.Symbol, Status: string(res.Status)}, nil
}

func (a *BinanceAdapter) CreateTakeProfitOrder(ctx context.Context, p port.StopOrderParams) (*port.OrderResult, error) {
	res, err := a.spot.NewCreateOrderService().
		Symbol(p.Symbol).
		Side(binance.SideType(p.Side)).
		Type(binance.OrderTypeTakeProfit).
		Quantity(p.Quantity).
		StopPrice(p.StopPrice).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.OrderResult{OrderID: res.OrderID, Symbol: res.Symbol, Status: string(res.Status)}, nil
}

func (a *BinanceAdapter) CreateStopLimitOrder(ctx context.Context, p port.StopLimitOrderParams) (*port.OrderResult, error) {
	res, err := a.spot.NewCreateOrderService().
		Symbol(p.Symbol).
		Side(binance.SideType(p.Side)).
		Type(binance.OrderTypeStopLossLimit).
		Quantity(p.Quantity).
		StopPrice(p.StopPrice).
		Price(p.LimitPrice).
		TimeInForce(binance.TimeInForceTypeGTC).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.OrderResult{OrderID: res.OrderID, Symbol: res.Symbol, Status: string(res.Status)}, nil
}

func (a *BinanceAdapter) CreateTrailingStopOrder(ctx context.Context, p port.TrailingStopOrderParams) (*port.OrderResult, error) {
	// callbackRate is a percentage (e.g. 1.5 for 1.5%); the Binance API
	// expects trailingDelta as an integer in BIPS (1% = 100 BIPS). The
	// exchange's TRAILING_DELTA filter caps at 2000 BIPS (20%).
	if p.CallbackRate < 0.1 || p.CallbackRate > 20 {
		return nil, fmt.Errorf("callbackRate must be between 0.1 and 20 (percent), got %g", p.CallbackRate)
	}
	bips := int(math.Round(p.CallbackRate * 100))
	res, err := a.spot.NewCreateOrderService().
		Symbol(p.Symbol).
		Side(binance.SideType(p.Side)).
		Type(binance.OrderTypeStopLoss).
		Quantity(p.Quantity).
		TrailingDelta(strconv.Itoa(bips)).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.OrderResult{OrderID: res.OrderID, Symbol: res.Symbol, Status: string(res.Status)}, nil
}

func (a *BinanceAdapter) CreateOCOOrder(ctx context.Context, p port.OCOOrderParams) (*port.OCOResult, error) {
	svc := a.spot.NewCreateOCOService().
		Symbol(p.Symbol).
		Side(binance.SideType(p.Side)).
		Quantity(p.Quantity).
		Price(p.Price).
		StopPrice(p.StopPrice)
	if p.StopLimitPrice != "" {
		svc = svc.StopLimitPrice(p.StopLimitPrice).
			StopLimitTimeInForce(binance.TimeInForceTypeGTC)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	orders := make([]port.Order, len(res.Orders))
	for i, o := range res.Orders {
		orders[i] = port.Order{OrderID: o.OrderID, Symbol: o.Symbol}
	}
	return &port.OCOResult{OrderListID: res.OrderListID, Symbol: p.Symbol, Orders: orders}, nil
}

// ─── Market Data ─────────────────────────────────────────────────────────────

func (a *BinanceAdapter) GetTicker(ctx context.Context, symbol string) (*port.Ticker, error) {
	res, err := a.spot.NewListPriceChangeStatsService().Symbol(symbol).Do(ctx)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("symbol %s not found", symbol)
	}
	r := res[0]
	return &port.Ticker{
		Symbol:             r.Symbol,
		PriceChange:        r.PriceChange,
		PriceChangePercent: r.PriceChangePercent,
		LastPrice:          r.LastPrice,
		Volume:             r.Volume,
	}, nil
}

func (a *BinanceAdapter) GetOrderBook(ctx context.Context, symbol string, limit int) (*port.OrderBook, error) {
	svc := a.spot.NewDepthService().Symbol(symbol)
	if limit > 0 {
		svc = svc.Limit(limit)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	bids := make([][]string, len(res.Bids))
	for i, b := range res.Bids {
		bids[i] = []string{b.Price, b.Quantity}
	}
	asks := make([][]string, len(res.Asks))
	for i, a := range res.Asks {
		asks[i] = []string{a.Price, a.Quantity}
	}
	return &port.OrderBook{LastUpdateID: res.LastUpdateID, Bids: bids, Asks: asks}, nil
}

func (a *BinanceAdapter) GetKlines(ctx context.Context, symbol, interval string, limit int) ([]port.Kline, error) {
	svc := a.spot.NewKlinesService().Symbol(symbol).Interval(interval)
	if limit > 0 {
		svc = svc.Limit(limit)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]port.Kline, len(res))
	for i, k := range res {
		out[i] = port.Kline{
			OpenTime:  k.OpenTime,
			Open:      k.Open,
			High:      k.High,
			Low:       k.Low,
			Close:     k.Close,
			Volume:    k.Volume,
			CloseTime: k.CloseTime,
		}
	}
	return out, nil
}

func (a *BinanceAdapter) GetFundingRate(ctx context.Context, symbol string) (*port.FundingRate, error) {
	svc := a.futures.NewPremiumIndexService()
	if symbol != "" {
		svc = svc.Symbol(symbol)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, fmt.Errorf("symbol %s not found", symbol)
	}
	return &port.FundingRate{
		Symbol:      res[0].Symbol,
		FundingRate: res[0].LastFundingRate,
		FundingTime: res[0].NextFundingTime,
	}, nil
}

// ─── Options ─────────────────────────────────────────────────────────────────

// optionsGuard rejects options calls outside production: Binance does not
// offer a public testnet for the European Options API (eapi.binance.com).
func (a *BinanceAdapter) optionsGuard() error {
	if a.env != "production" {
		return fmt.Errorf("options tools are unavailable in %s mode: Binance has no public options testnet (set BINANCE_TESTNET=false to trade options against production)", a.env)
	}
	return nil
}

func (a *BinanceAdapter) CreateOptionOrder(ctx context.Context, p port.OptionOrderParams) (*port.OptionOrder, error) {
	if err := a.optionsGuard(); err != nil {
		return nil, err
	}
	// The options API only supports LIMIT orders here, which require a price.
	// Reject early with a clear message instead of forwarding an order Binance
	// will refuse.
	if p.Price == "" {
		return nil, fmt.Errorf("price is required for option orders")
	}
	res, err := a.opts.NewCreateOrderService().
		Symbol(p.Symbol).
		Side(options.SideType(p.Side)).
		Type(options.OrderType("LIMIT")).
		Quantity(p.Quantity).
		Price(p.Price).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.OptionOrder{
		OrderID:  strconv.FormatInt(res.OrderId, 10),
		Symbol:   res.Symbol,
		Status:   string(res.Status),
		Price:    res.Price,
		Quantity: res.Quantity,
	}, nil
}

func (a *BinanceAdapter) GetOptionChain(ctx context.Context, underlying string) ([]port.OptionSymbol, error) {
	if err := a.optionsGuard(); err != nil {
		return nil, err
	}
	res, err := a.opts.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return nil, err
	}
	var out []port.OptionSymbol
	for _, s := range res.OptionSymbols {
		if underlying == "" || s.Underlying == underlying {
			out = append(out, port.OptionSymbol{
				Symbol:      s.Symbol,
				Underlying:  s.Underlying,
				StrikePrice: s.StrikePrice,
				ExpiryDate:  s.ExpiryDate,
			})
		}
	}
	return out, nil
}

func (a *BinanceAdapter) GetOptionPositions(ctx context.Context) ([]port.OptionPosition, error) {
	if err := a.optionsGuard(); err != nil {
		return nil, err
	}
	res, err := a.opts.NewPositionService().Do(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]port.OptionPosition, len(res))
	for i, p := range res {
		out[i] = port.OptionPosition{
			Symbol:        p.Symbol,
			PositionAmt:   p.Quantity,
			AvgPrice:      p.EntryPrice,
			UnrealizedPnL: p.UnrealizedPNL,
		}
	}
	return out, nil
}

func (a *BinanceAdapter) GetOptionInfo(ctx context.Context, symbol string) (*port.OptionSymbol, error) {
	if err := a.optionsGuard(); err != nil {
		return nil, err
	}
	res, err := a.opts.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range res.OptionSymbols {
		if s.Symbol == symbol {
			return &port.OptionSymbol{
				Symbol:      s.Symbol,
				Underlying:  s.Underlying,
				StrikePrice: s.StrikePrice,
				ExpiryDate:  s.ExpiryDate,
			}, nil
		}
	}
	return nil, fmt.Errorf("option symbol %s not found", symbol)
}

// ─── Futures ─────────────────────────────────────────────────────────────────

func (a *BinanceAdapter) CreateContractOrder(ctx context.Context, p port.ContractOrderParams) (*port.OrderResult, error) {
	// Binance rejects conditional types on /fapi/v1/order with -4120: they
	// moved to the Algo Order API (/fapi/v1/algoOrder, algoType=CONDITIONAL).
	isTrigger := p.Type == "STOP_MARKET" || p.Type == "TAKE_PROFIT_MARKET"
	if p.ClosePosition && !isTrigger {
		return nil, fmt.Errorf("closePosition is only valid for STOP_MARKET and TAKE_PROFIT_MARKET orders")
	}
	if isTrigger {
		return a.createFuturesAlgoOrder(ctx, p)
	}
	if p.Quantity == "" {
		return nil, fmt.Errorf("quantity is required unless closePosition is true")
	}
	svc := a.futures.NewCreateOrderService().
		Symbol(p.Symbol).
		Side(futures.SideType(p.Side)).
		Type(futures.OrderType(p.Type)).
		Quantity(p.Quantity)
	if p.ReduceOnly {
		svc = svc.ReduceOnly(true)
	}
	if p.Price != "" {
		svc = svc.Price(p.Price)
	}
	if p.Type == "LIMIT" {
		svc = svc.TimeInForce(futures.TimeInForceTypeGTC)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.OrderResult{OrderID: res.OrderID, Symbol: res.Symbol, Status: string(res.Status)}, nil
}

// createFuturesAlgoOrder places STOP_MARKET / TAKE_PROFIT_MARKET via the Algo
// Order API. The result carries AlgoID (not OrderID): algo orders live in
// their own namespace and are managed via GetOpenAlgoOrders/CancelAlgoOrder.
func (a *BinanceAdapter) createFuturesAlgoOrder(ctx context.Context, p port.ContractOrderParams) (*port.OrderResult, error) {
	if p.StopPrice == "" {
		return nil, fmt.Errorf("stopPrice is required for %s orders", p.Type)
	}
	if !p.ClosePosition && p.Quantity == "" {
		return nil, fmt.Errorf("quantity is required unless closePosition is true")
	}
	svc := a.futures.NewCreateAlgoOrderService().
		AlgoType(futures.OrderAlgoTypeConditional).
		Symbol(p.Symbol).
		Side(futures.SideType(p.Side)).
		Type(futures.AlgoOrderType(p.Type)).
		TriggerPrice(p.StopPrice)
	if p.ClosePosition {
		// closePosition closes the whole position at trigger; Binance rejects
		// quantity and reduceOnly alongside it.
		svc = svc.ClosePosition(true)
	} else {
		svc = svc.Quantity(p.Quantity)
		if p.ReduceOnly {
			svc = svc.ReduceOnly(true)
		}
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.OrderResult{AlgoID: res.AlgoId, Symbol: res.Symbol, Status: string(res.AlgoStatus)}, nil
}

func (a *BinanceAdapter) GetOpenAlgoOrders(ctx context.Context, symbol string) ([]port.AlgoOrder, error) {
	svc := a.futures.NewListOpenAlgoOrdersService()
	if symbol != "" {
		svc = svc.Symbol(symbol)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]port.AlgoOrder, len(res))
	for i, o := range res {
		out[i] = port.AlgoOrder{
			AlgoID:        o.AlgoId,
			Symbol:        o.Symbol,
			Side:          string(o.Side),
			Type:          string(o.OrderType),
			Status:        string(o.AlgoStatus),
			TriggerPrice:  o.TriggerPrice,
			Quantity:      o.Quantity,
			ReduceOnly:    o.ReduceOnly,
			ClosePosition: o.ClosePosition,
		}
	}
	return out, nil
}

func (a *BinanceAdapter) CancelAlgoOrder(ctx context.Context, algoID int64) (*port.AlgoCancelResult, error) {
	if algoID <= 0 {
		return nil, fmt.Errorf("algoId must be a positive integer, got %d", algoID)
	}
	res, err := a.futures.NewCancelAlgoOrderService().AlgoID(algoID).Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.AlgoCancelResult{
		AlgoID:  int64(res.AlgoId),
		Code:    res.Code,
		Message: res.Message,
	}, nil
}

func (a *BinanceAdapter) ClosePosition(ctx context.Context, symbol string) (*port.OrderResult, error) {
	// The contract only gives us a symbol, so we must look up the live position
	// to know which way to close: a long is closed with a SELL, a short with a
	// BUY. PositionAmt is signed (positive = long, negative = short).
	risks, err := a.futures.NewGetPositionRiskService().Symbol(symbol).Do(ctx)
	if err != nil {
		return nil, err
	}

	var pos *futures.PositionRisk
	for i := range risks {
		if risks[i].Symbol != symbol {
			continue
		}
		amt, perr := strconv.ParseFloat(risks[i].PositionAmt, 64)
		if perr == nil && amt != 0 {
			pos = risks[i]
			break
		}
	}
	if pos == nil {
		return nil, fmt.Errorf("no open position for %s", symbol)
	}

	amt, err := strconv.ParseFloat(pos.PositionAmt, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid position amount %q: %w", pos.PositionAmt, err)
	}

	side := futures.SideTypeSell // long → sell to close
	if amt < 0 {
		side = futures.SideTypeBuy // short → buy to close
		amt = -amt
	}
	quantity := strconv.FormatFloat(amt, 'f', -1, 64)

	svc := a.futures.NewCreateOrderService().
		Symbol(symbol).
		Side(side).
		Type(futures.OrderTypeMarket).
		Quantity(quantity)

	// In hedge mode the position is tagged LONG/SHORT and must be addressed by
	// PositionSide; in one-way mode it is BOTH and ReduceOnly guards against
	// accidentally flipping into an opposite position.
	if pos.PositionSide == "LONG" || pos.PositionSide == "SHORT" {
		svc = svc.PositionSide(futures.PositionSideType(pos.PositionSide))
	} else {
		svc = svc.ReduceOnly(true)
	}

	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.OrderResult{OrderID: res.OrderID, Symbol: res.Symbol, Status: string(res.Status)}, nil
}

func (a *BinanceAdapter) GetFuturesPositions(ctx context.Context, symbol string) ([]port.FuturesPosition, error) {
	svc := a.futures.NewGetPositionRiskService()
	if symbol != "" {
		svc = svc.Symbol(symbol)
	}
	res, err := svc.Do(ctx)
	if err != nil {
		return nil, err
	}
	var out []port.FuturesPosition
	for _, p := range res {
		if p.PositionAmt != "0" && p.PositionAmt != "0.000" {
			out = append(out, port.FuturesPosition{
				Symbol:           p.Symbol,
				PositionAmt:      p.PositionAmt,
				EntryPrice:       p.EntryPrice,
				UnRealizedProfit: p.UnRealizedProfit,
				LiquidationPrice: p.LiquidationPrice,
				Leverage:         p.Leverage,
				PositionSide:     p.PositionSide,
			})
		}
	}
	return out, nil
}

// ─── Account ─────────────────────────────────────────────────────────────────

func (a *BinanceAdapter) GetBalance(ctx context.Context, asset string) ([]port.Balance, error) {
	res, err := a.spot.NewGetAccountService().OmitZeroBalances(true).Do(ctx)
	if err != nil {
		return nil, err
	}
	var out []port.Balance
	for _, b := range res.Balances {
		if asset != "" && b.Asset != asset {
			continue
		}
		out = append(out, port.Balance{Asset: b.Asset, Free: b.Free, Locked: b.Locked})
	}
	return out, nil
}

func (a *BinanceAdapter) GetPositions(ctx context.Context) ([]port.FuturesPosition, error) {
	return a.GetFuturesPositions(ctx, "")
}

func (a *BinanceAdapter) GetFuturesBalance(ctx context.Context, asset string) ([]port.FuturesBalance, error) {
	res, err := a.futures.NewGetBalanceService().Do(ctx)
	if err != nil {
		return nil, err
	}
	var out []port.FuturesBalance
	for _, b := range res {
		if asset != "" && b.Asset != asset {
			continue
		}
		// With no asset filter, mirror the spot tool and omit empty wallets.
		if asset == "" && isZero(b.Balance) && isZero(b.CrossUnPnl) {
			continue
		}
		out = append(out, port.FuturesBalance{
			Asset:            b.Asset,
			Balance:          b.Balance,
			CrossUnPnl:       b.CrossUnPnl,
			AvailableBalance: b.AvailableBalance,
		})
	}
	return out, nil
}

// isZero reports whether a Binance decimal string is numerically zero.
func isZero(s string) bool {
	f, err := strconv.ParseFloat(s, 64)
	return err == nil && f == 0
}

func (a *BinanceAdapter) SetLeverage(ctx context.Context, p port.SetLeverageParams) error {
	_, err := a.futures.NewChangeLeverageService().
		Symbol(p.Symbol).
		Leverage(p.Leverage).
		Do(ctx)
	return err
}

func (a *BinanceAdapter) SetMarginMode(ctx context.Context, p port.SetMarginModeParams) error {
	return a.futures.NewChangeMarginTypeService().
		Symbol(p.Symbol).
		MarginType(futures.MarginType(p.MarginMode)).
		Do(ctx)
}

func (a *BinanceAdapter) TransferFunds(ctx context.Context, p port.TransferFundsParams) error {
	transferType, err := resolveTransferType(p.FromAccount, p.ToAccount)
	if err != nil {
		return err
	}
	_, err = a.spot.NewUserUniversalTransferService().
		Type(transferType).
		Asset(p.Asset).
		Amount(p.Amount).
		Do(ctx)
	return err
}

func resolveTransferType(from, to string) (binance.UserUniversalTransferType, error) {
	key := from + "_" + to
	m := map[string]binance.UserUniversalTransferType{
		"SPOT_FUTURES":    binance.UserUniversalTransferTypeMainToUmFutures,
		"SPOT_MARGIN":     binance.UserUniversalTransferTypeMainToMargin,
		"SPOT_FUNDING":    binance.UserUniversalTransferTypeMainToFunding,
		"FUTURES_SPOT":    binance.UserUniversalTransferTypeUmFuturesToMain,
		"FUTURES_MARGIN":  binance.UserUniversalTransferTypeUmFuturesToMargin,
		"FUTURES_FUNDING": binance.UserUniversalTransferTypeUmFuturesToFunding,
		"MARGIN_SPOT":     binance.UserUniversalTransferTypeMarginToMain,
		"MARGIN_FUTURES":  binance.UserUniversalTransferTypeMarginToUmFutures,
		"MARGIN_FUNDING":  binance.UserUniversalTransferTypeMarginToFunding,
		"FUNDING_SPOT":    binance.UserUniversalTransferTypeFundingToMain,
		"FUNDING_FUTURES": binance.UserUniversalTransferTypeFundingToUmFutures,
		"FUNDING_MARGIN":  binance.UserUniversalTransferTypeFundingToMargin,
	}
	t, ok := m[key]
	if !ok {
		return "", fmt.Errorf("unsupported transfer: %s → %s", from, to)
	}
	return t, nil
}

// ─── System ──────────────────────────────────────────────────────────────────

func (a *BinanceAdapter) GetServerInfo(ctx context.Context) (*port.ServerInfo, error) {
	t, err := a.spot.NewServerTimeService().Do(ctx)
	if err != nil {
		return nil, err
	}
	return &port.ServerInfo{
		ServerTime:  t,
		Version:     "1.0.0",
		Environment: a.env,
	}, nil
}

// compile-time check
var _ port.BinancePort = (*BinanceAdapter)(nil)
