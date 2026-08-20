package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// callTool is a test helper: registers tools then invokes the named one with args.
func callTool(t *testing.T, b port.BinancePort, toolName string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	s := server.NewMCPServer("test", "0.0.1")
	RegisterAll(s, b)

	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      toolName,
			"arguments": args,
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	msg := s.HandleMessage(context.Background(), raw)
	resp, ok := msg.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("expected JSONRPCResponse, got %T: %+v", msg, msg)
	}
	result, ok := resp.Result.(*mcp.CallToolResult)
	if !ok {
		t.Fatalf("expected *mcp.CallToolResult in result, got %T: %+v", resp.Result, resp.Result)
	}
	return result
}

func assertOK(t *testing.T, res *mcp.CallToolResult) {
	t.Helper()
	if res.IsError {
		t.Fatalf("expected success, got error: %s", firstText(res))
	}
}

func assertError(t *testing.T, res *mcp.CallToolResult, contains string) {
	t.Helper()
	if !res.IsError {
		t.Fatalf("expected error result, got success: %s", firstText(res))
	}
	if contains != "" && !strings.Contains(firstText(res), contains) {
		t.Fatalf("error text %q does not contain %q", firstText(res), contains)
	}
}

func firstText(res *mcp.CallToolResult) string {
	for _, c := range res.Content {
		switch tc := c.(type) {
		case mcp.TextContent:
			return tc.Text
		case *mcp.TextContent:
			return tc.Text
		}
	}
	return ""
}

// ─── Spot Trading ─────────────────────────────────────────────────────────────

func TestCreateSpotOrder_Success(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 42, Symbol: "BTCUSDT", Status: "NEW"}}
	res := callTool(t, m, "create_spot_order", map[string]any{
		"symbol": "BTCUSDT", "side": "BUY", "type": "MARKET", "quantity": 0.001,
	})
	assertOK(t, res)
	if m.lastCreateSpotOrder.Symbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %s", m.lastCreateSpotOrder.Symbol)
	}
}

func TestCreateSpotOrder_Error(t *testing.T) {
	m := &mockPort{errSpotOrder: sentinelErr}
	res := callTool(t, m, "create_spot_order", map[string]any{
		"symbol": "BTCUSDT", "side": "BUY", "type": "MARKET", "quantity": 0.001,
	})
	assertError(t, res, "create_spot_order failed")
}

func TestCreateSpotOrder_WithPrice(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 1, Symbol: "ETHUSDT", Status: "NEW"}}
	res := callTool(t, m, "create_spot_order", map[string]any{
		"symbol": "ETHUSDT", "side": "BUY", "type": "LIMIT", "quantity": 1.0, "price": 3000.0,
	})
	assertOK(t, res)
	if m.lastCreateSpotOrder.Price != "3000" {
		t.Errorf("expected price 3000, got %s", m.lastCreateSpotOrder.Price)
	}
}

// ─── Order Management ─────────────────────────────────────────────────────────

func TestCancelOrder_Success(t *testing.T) {
	m := &mockPort{order: &port.Order{OrderID: 99, Symbol: "BTCUSDT", Status: "CANCELED"}}
	res := callTool(t, m, "cancel_order", map[string]any{"symbol": "BTCUSDT", "orderId": float64(99)})
	assertOK(t, res)
	if m.lastCancelOrder.OrderID != 99 {
		t.Errorf("expected orderID 99, got %d", m.lastCancelOrder.OrderID)
	}
	if m.lastCancelOrder.Market != "spot" {
		t.Errorf("expected market spot (default), got %s", m.lastCancelOrder.Market)
	}
}

func TestCancelOrder_Futures(t *testing.T) {
	m := &mockPort{order: &port.Order{OrderID: 42, Symbol: "BTCUSDT", Status: "CANCELED"}}
	res := callTool(t, m, "cancel_order", map[string]any{
		"symbol": "BTCUSDT", "orderId": float64(42), "market": "futures",
	})
	assertOK(t, res)
	if m.lastCancelOrder.Market != "futures" {
		t.Errorf("expected market futures, got %s", m.lastCancelOrder.Market)
	}
}

func TestCancelOrder_Error(t *testing.T) {
	m := &mockPort{errCancelOrder: sentinelErr}
	res := callTool(t, m, "cancel_order", map[string]any{"symbol": "BTCUSDT", "orderId": float64(1)})
	assertError(t, res, "cancel_order failed")
}

func TestGetOpenOrders_Success(t *testing.T) {
	m := &mockPort{orders: []port.Order{{OrderID: 1, Symbol: "BTCUSDT"}}}
	res := callTool(t, m, "get_open_orders", map[string]any{"symbol": "BTCUSDT"})
	assertOK(t, res)
}

func TestGetOpenOrders_NoSymbol(t *testing.T) {
	m := &mockPort{orders: nil}
	res := callTool(t, m, "get_open_orders", map[string]any{})
	assertOK(t, res)
	if m.lastGetOpenOrders.Symbol != "" {
		t.Errorf("expected empty symbol, got %s", m.lastGetOpenOrders.Symbol)
	}
	if m.lastGetOpenOrders.Market != "spot" {
		t.Errorf("expected market spot (default), got %s", m.lastGetOpenOrders.Market)
	}
}

func TestGetOpenOrders_Futures(t *testing.T) {
	m := &mockPort{orders: []port.Order{{OrderID: 7, Symbol: "BTCUSDT"}}}
	res := callTool(t, m, "get_open_orders", map[string]any{"symbol": "BTCUSDT", "market": "futures"})
	assertOK(t, res)
	if m.lastGetOpenOrders.Market != "futures" {
		t.Errorf("expected market futures, got %s", m.lastGetOpenOrders.Market)
	}
}

func TestGetOpenOrders_Error(t *testing.T) {
	m := &mockPort{errOpenOrders: sentinelErr}
	res := callTool(t, m, "get_open_orders", map[string]any{})
	assertError(t, res, "get_open_orders failed")
}

func TestGetOrderStatus_Success(t *testing.T) {
	m := &mockPort{order: &port.Order{OrderID: 5, Symbol: "ETHUSDT", Status: "FILLED"}}
	res := callTool(t, m, "get_order_status", map[string]any{"symbol": "ETHUSDT", "orderId": float64(5)})
	assertOK(t, res)
	if m.lastGetOrderStatus.Market != "spot" {
		t.Errorf("expected market spot (default), got %s", m.lastGetOrderStatus.Market)
	}
}

func TestGetOrderStatus_Futures(t *testing.T) {
	m := &mockPort{order: &port.Order{OrderID: 8, Symbol: "BTCUSDT", Status: "NEW"}}
	res := callTool(t, m, "get_order_status", map[string]any{
		"symbol": "BTCUSDT", "orderId": float64(8), "market": "futures",
	})
	assertOK(t, res)
	if m.lastGetOrderStatus.Market != "futures" {
		t.Errorf("expected market futures, got %s", m.lastGetOrderStatus.Market)
	}
}

func TestGetOrderStatus_Error(t *testing.T) {
	m := &mockPort{errOrderStatus: sentinelErr}
	res := callTool(t, m, "get_order_status", map[string]any{"symbol": "ETHUSDT", "orderId": float64(5)})
	assertError(t, res, "get_order_status failed")
}

func TestGetMyTrades_Success(t *testing.T) {
	m := &mockPort{trades: []port.Trade{{ID: 1, Symbol: "BTCUSDT", Price: "50000", Qty: "0.001", IsBuyer: true}}}
	res := callTool(t, m, "get_my_trades", map[string]any{"symbol": "BTCUSDT", "limit": float64(10)})
	assertOK(t, res)
	if m.lastGetMyTrades.Limit != 10 {
		t.Errorf("expected limit 10, got %d", m.lastGetMyTrades.Limit)
	}
}

func TestGetMyTrades_DefaultLimit(t *testing.T) {
	m := &mockPort{trades: nil}
	callTool(t, m, "get_my_trades", map[string]any{"symbol": "BTCUSDT"})
	if m.lastGetMyTrades.Limit != 100 {
		t.Errorf("expected default limit 100, got %d", m.lastGetMyTrades.Limit)
	}
}

func TestGetMyTrades_DefaultMarketIsSpot(t *testing.T) {
	m := &mockPort{trades: nil}
	callTool(t, m, "get_my_trades", map[string]any{"symbol": "BTCUSDT"})
	if m.lastGetMyTrades.Market != "spot" {
		t.Errorf("expected market spot (default), got %s", m.lastGetMyTrades.Market)
	}
}

func TestGetMyTrades_Futures(t *testing.T) {
	m := &mockPort{trades: []port.Trade{{ID: 7, Symbol: "BTCUSDT", Side: "SELL", RealizedPnl: "-0.91"}}}
	res := callTool(t, m, "get_my_trades", map[string]any{"symbol": "BTCUSDT", "market": "futures"})
	assertOK(t, res)
	if m.lastGetMyTrades.Market != "futures" {
		t.Errorf("expected market futures, got %s", m.lastGetMyTrades.Market)
	}
	if !strings.Contains(firstText(res), "realizedPnl") {
		t.Error("expected realizedPnl in futures trade response")
	}
}

func TestGetMyTrades_Error(t *testing.T) {
	m := &mockPort{errMyTrades: sentinelErr}
	res := callTool(t, m, "get_my_trades", map[string]any{"symbol": "BTCUSDT"})
	assertError(t, res, "get_my_trades failed")
}

func TestCancelAllOrders_Success(t *testing.T) {
	m := &mockPort{orders: []port.Order{{OrderID: 1}, {OrderID: 2}}}
	res := callTool(t, m, "cancel_all_orders", map[string]any{"symbol": "BTCUSDT"})
	assertOK(t, res)
	if m.lastCancelAllOrders.Symbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %s", m.lastCancelAllOrders.Symbol)
	}
	if m.lastCancelAllOrders.Market != "spot" {
		t.Errorf("expected market spot (default), got %s", m.lastCancelAllOrders.Market)
	}
}

func TestCancelAllOrders_Futures(t *testing.T) {
	m := &mockPort{orders: []port.Order{{Symbol: "BTCUSDT", Status: "CANCELED"}}}
	res := callTool(t, m, "cancel_all_orders", map[string]any{
		"symbol": "BTCUSDT", "market": "futures",
	})
	assertOK(t, res)
	if m.lastCancelAllOrders.Market != "futures" {
		t.Errorf("expected market futures, got %s", m.lastCancelAllOrders.Market)
	}
}

func TestCancelAllOrders_Error(t *testing.T) {
	m := &mockPort{errCancelAll: sentinelErr}
	res := callTool(t, m, "cancel_all_orders", map[string]any{"symbol": "BTCUSDT"})
	assertError(t, res, "cancel_all_orders failed")
}

// ─── Risk Control ─────────────────────────────────────────────────────────────

func TestCreateStopLossOrder_Success(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 10, Symbol: "BTCUSDT", Status: "NEW"}}
	res := callTool(t, m, "create_stop_loss_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1, "stopPrice": 45000.0,
	})
	assertOK(t, res)
}

func TestCreateStopLossOrder_Error(t *testing.T) {
	m := &mockPort{errStopLoss: sentinelErr}
	res := callTool(t, m, "create_stop_loss_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1, "stopPrice": 45000.0,
	})
	assertError(t, res, "create_stop_loss_order failed")
}

func TestCreateTakeProfitOrder_Success(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 11, Symbol: "BTCUSDT", Status: "NEW"}}
	res := callTool(t, m, "create_take_profit_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1, "stopPrice": 55000.0,
	})
	assertOK(t, res)
}

func TestCreateTakeProfitOrder_Error(t *testing.T) {
	m := &mockPort{errTakeProfit: sentinelErr}
	res := callTool(t, m, "create_take_profit_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1, "stopPrice": 55000.0,
	})
	assertError(t, res, "create_take_profit_order failed")
}

func TestCreateStopLimitOrder_Success(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 12}}
	res := callTool(t, m, "create_stop_limit_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1, "stopPrice": 45000.0, "limitPrice": 44900.0,
	})
	assertOK(t, res)
}

func TestCreateStopLimitOrder_Error(t *testing.T) {
	m := &mockPort{errStopLimit: sentinelErr}
	res := callTool(t, m, "create_stop_limit_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1, "stopPrice": 45000.0, "limitPrice": 44900.0,
	})
	assertError(t, res, "create_stop_limit_order failed")
}

func TestCreateTrailingStopOrder_Success(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 13}}
	res := callTool(t, m, "create_trailing_stop_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1, "callbackRate": 1.5,
	})
	assertOK(t, res)
	if m.lastTrailingStop.CallbackRate != 1.5 {
		t.Errorf("expected callbackRate 1.5, got %g", m.lastTrailingStop.CallbackRate)
	}
}

func TestCreateTrailingStopOrder_Error(t *testing.T) {
	m := &mockPort{errTrailingStop: sentinelErr}
	res := callTool(t, m, "create_trailing_stop_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1, "callbackRate": 1.0,
	})
	assertError(t, res, "create_trailing_stop_order failed")
}

func TestCreateOCOOrder_Success(t *testing.T) {
	m := &mockPort{ocoResult: &port.OCOResult{OrderListID: 1, Symbol: "BTCUSDT"}}
	res := callTool(t, m, "create_oco_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1, "price": 55000.0, "stopPrice": 45000.0,
	})
	assertOK(t, res)
}

func TestCreateOCOOrder_WithStopLimitPrice(t *testing.T) {
	m := &mockPort{ocoResult: &port.OCOResult{OrderListID: 2, Symbol: "BTCUSDT"}}
	res := callTool(t, m, "create_oco_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1,
		"price": 55000.0, "stopPrice": 45000.0, "stopLimitPrice": 44900.0,
	})
	assertOK(t, res)
}

func TestCreateOCOOrder_Error(t *testing.T) {
	m := &mockPort{errOCO: sentinelErr}
	res := callTool(t, m, "create_oco_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.1, "price": 55000.0, "stopPrice": 45000.0,
	})
	assertError(t, res, "create_oco_order failed")
}

// ─── Market Data ──────────────────────────────────────────────────────────────

func TestGetTicker_Success(t *testing.T) {
	m := &mockPort{ticker: &port.Ticker{Symbol: "BTCUSDT", LastPrice: "50000"}}
	res := callTool(t, m, "get_ticker", map[string]any{"symbol": "BTCUSDT"})
	assertOK(t, res)
	if !strings.Contains(firstText(res), "BTCUSDT") {
		t.Error("expected BTCUSDT in response")
	}
}

func TestGetTicker_Error(t *testing.T) {
	m := &mockPort{errTicker: sentinelErr}
	res := callTool(t, m, "get_ticker", map[string]any{"symbol": "BTCUSDT"})
	assertError(t, res, "get_ticker failed")
}

func TestGetOrderBook_Success(t *testing.T) {
	m := &mockPort{orderBook: &port.OrderBook{LastUpdateID: 1, Bids: [][]string{{"50000", "0.1"}}, Asks: [][]string{{"50001", "0.2"}}}}
	res := callTool(t, m, "get_order_book", map[string]any{"symbol": "BTCUSDT"})
	assertOK(t, res)
}

func TestGetOrderBook_Error(t *testing.T) {
	m := &mockPort{errOrderBook: sentinelErr}
	res := callTool(t, m, "get_order_book", map[string]any{"symbol": "BTCUSDT"})
	assertError(t, res, "get_order_book failed")
}

func TestGetKlines_Success(t *testing.T) {
	m := &mockPort{klines: []port.Kline{{OpenTime: 1000, Open: "50000", Close: "51000"}}}
	res := callTool(t, m, "get_klines", map[string]any{"symbol": "BTCUSDT", "interval": "1h"})
	assertOK(t, res)
}

func TestGetKlines_Error(t *testing.T) {
	m := &mockPort{errKlines: sentinelErr}
	res := callTool(t, m, "get_klines", map[string]any{"symbol": "BTCUSDT", "interval": "1h"})
	assertError(t, res, "get_klines failed")
}

func TestGetFundingRate_Success(t *testing.T) {
	m := &mockPort{fundingRate: &port.FundingRate{Symbol: "BTCUSDT", FundingRate: "0.0001"}}
	res := callTool(t, m, "get_funding_rate", map[string]any{"symbol": "BTCUSDT"})
	assertOK(t, res)
}

func TestGetFundingRate_Error(t *testing.T) {
	m := &mockPort{errFundingRate: sentinelErr}
	res := callTool(t, m, "get_funding_rate", map[string]any{"symbol": "BTCUSDT"})
	assertError(t, res, "get_funding_rate failed")
}

// ─── Options ──────────────────────────────────────────────────────────────────

func TestCreateOptionOrder_Success(t *testing.T) {
	m := &mockPort{optionOrder: &port.OptionOrder{OrderID: "opt1", Symbol: "BTC-250101-50000-C"}}
	res := callTool(t, m, "create_option_order", map[string]any{
		"symbol": "BTC-250101-50000-C", "side": "BUY", "quantity": 1.0, "price": 500.0,
	})
	assertOK(t, res)
}

func TestCreateOptionOrder_Error(t *testing.T) {
	m := &mockPort{errOptionOrder: sentinelErr}
	res := callTool(t, m, "create_option_order", map[string]any{
		"symbol": "BTC-250101-50000-C", "side": "BUY", "quantity": 1.0, "price": 500.0,
	})
	assertError(t, res, "create_option_order failed")
}

func TestGetOptionChain_Success(t *testing.T) {
	m := &mockPort{optionSymbols: []port.OptionSymbol{{Symbol: "BTC-250101-50000-C", Underlying: "BTCUSDT"}}}
	res := callTool(t, m, "get_option_chain", map[string]any{"underlying": "BTCUSDT"})
	assertOK(t, res)
}

func TestGetOptionChain_Error(t *testing.T) {
	m := &mockPort{errOptionChain: sentinelErr}
	res := callTool(t, m, "get_option_chain", map[string]any{"underlying": "BTCUSDT"})
	assertError(t, res, "get_option_chain failed")
}

func TestGetOptionPositions_Success(t *testing.T) {
	m := &mockPort{optionPositions: []port.OptionPosition{{Symbol: "BTC-250101-50000-C", PositionAmt: "1"}}}
	res := callTool(t, m, "get_option_positions", map[string]any{})
	assertOK(t, res)
}

func TestGetOptionPositions_Error(t *testing.T) {
	m := &mockPort{errOptionPositions: sentinelErr}
	res := callTool(t, m, "get_option_positions", map[string]any{})
	assertError(t, res, "get_option_positions failed")
}

func TestGetOptionInfo_Success(t *testing.T) {
	m := &mockPort{optionSymbol: &port.OptionSymbol{Symbol: "BTC-250101-50000-C", Underlying: "BTCUSDT"}}
	res := callTool(t, m, "get_option_info", map[string]any{"symbol": "BTC-250101-50000-C"})
	assertOK(t, res)
}

func TestGetOptionInfo_Error(t *testing.T) {
	m := &mockPort{errOptionInfo: sentinelErr}
	res := callTool(t, m, "get_option_info", map[string]any{"symbol": "BTC-250101-50000-C"})
	assertError(t, res, "get_option_info failed")
}

// ─── Futures ──────────────────────────────────────────────────────────────────

func TestCreateContractOrder_Success(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 20, Symbol: "BTCUSDT", Status: "NEW"}}
	res := callTool(t, m, "create_contract_order", map[string]any{
		"symbol": "BTCUSDT", "side": "BUY", "type": "MARKET", "quantity": 0.01,
	})
	assertOK(t, res)
	if m.lastContractOrder.Symbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %s", m.lastContractOrder.Symbol)
	}
}

func TestCreateContractOrder_WithPrice(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 21}}
	res := callTool(t, m, "create_contract_order", map[string]any{
		"symbol": "BTCUSDT", "side": "BUY", "type": "LIMIT", "quantity": 0.01, "price": 50000.0,
	})
	assertOK(t, res)
	if m.lastContractOrder.Price != "50000" {
		t.Errorf("expected price 50000, got %s", m.lastContractOrder.Price)
	}
}

func TestCreateContractOrder_StopMarket(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 22, Symbol: "BTCUSDT", Status: "NEW"}}
	res := callTool(t, m, "create_contract_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "type": "STOP_MARKET",
		"quantity": 0.01, "stopPrice": 45000.0, "reduceOnly": true,
	})
	assertOK(t, res)
	if m.lastContractOrder.StopPrice != "45000" {
		t.Errorf("expected stopPrice 45000, got %s", m.lastContractOrder.StopPrice)
	}
	if !m.lastContractOrder.ReduceOnly {
		t.Error("expected reduceOnly true")
	}
	if m.lastContractOrder.ClosePosition {
		t.Error("expected closePosition false")
	}
}

func TestCreateContractOrder_TakeProfitMarketClosePosition(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 23, Symbol: "BTCUSDT", Status: "NEW"}}
	res := callTool(t, m, "create_contract_order", map[string]any{
		"symbol": "BTCUSDT", "side": "SELL", "type": "TAKE_PROFIT_MARKET",
		"stopPrice": 60000.0, "closePosition": true,
	})
	assertOK(t, res)
	if !m.lastContractOrder.ClosePosition {
		t.Error("expected closePosition true")
	}
	if m.lastContractOrder.Quantity != "" {
		t.Errorf("expected no quantity with closePosition, got %s", m.lastContractOrder.Quantity)
	}
	if m.lastContractOrder.StopPrice != "60000" {
		t.Errorf("expected stopPrice 60000, got %s", m.lastContractOrder.StopPrice)
	}
}

func TestCreateContractOrder_Error(t *testing.T) {
	m := &mockPort{errContractOrder: sentinelErr}
	res := callTool(t, m, "create_contract_order", map[string]any{
		"symbol": "BTCUSDT", "side": "BUY", "type": "MARKET", "quantity": 0.01,
	})
	assertError(t, res, "create_contract_order failed")
}

func TestGetOpenAlgoOrders_Success(t *testing.T) {
	m := &mockPort{algoOrders: []port.AlgoOrder{{AlgoID: 2146760, Symbol: "BTCUSDT", Type: "STOP_MARKET", Status: "NEW"}}}
	res := callTool(t, m, "get_open_algo_orders", map[string]any{"symbol": "BTCUSDT"})
	assertOK(t, res)
	if m.lastAlgoOrdersSymbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", m.lastAlgoOrdersSymbol)
	}
	if !strings.Contains(firstText(res), "2146760") {
		t.Error("expected algoId 2146760 in response")
	}
}

func TestGetOpenAlgoOrders_Error(t *testing.T) {
	m := &mockPort{errAlgoOrders: sentinelErr}
	res := callTool(t, m, "get_open_algo_orders", map[string]any{})
	assertError(t, res, "get_open_algo_orders failed")
}

func TestCancelAlgoOrder_Success(t *testing.T) {
	m := &mockPort{algoCancelResult: &port.AlgoCancelResult{AlgoID: 2146760, Code: "200", Message: "success"}}
	res := callTool(t, m, "cancel_algo_order", map[string]any{"algoId": float64(2146760)})
	assertOK(t, res)
	if m.lastCancelAlgoID != 2146760 {
		t.Errorf("expected algoId 2146760, got %d", m.lastCancelAlgoID)
	}
}

func TestCancelAlgoOrder_Error(t *testing.T) {
	m := &mockPort{errCancelAlgoOrder: sentinelErr}
	res := callTool(t, m, "cancel_algo_order", map[string]any{"algoId": float64(1)})
	assertError(t, res, "cancel_algo_order failed")
}

func TestClosePosition_Success(t *testing.T) {
	m := &mockPort{spotOrderResult: &port.OrderResult{OrderID: 30, Symbol: "BTCUSDT", Status: "NEW"}}
	res := callTool(t, m, "close_position", map[string]any{"symbol": "BTCUSDT"})
	assertOK(t, res)
	if m.lastClosePositionSymbol != "BTCUSDT" {
		t.Errorf("expected BTCUSDT, got %s", m.lastClosePositionSymbol)
	}
}

func TestClosePosition_Error(t *testing.T) {
	m := &mockPort{errClosePosition: sentinelErr}
	res := callTool(t, m, "close_position", map[string]any{"symbol": "BTCUSDT"})
	assertError(t, res, "close_position failed")
}

func TestGetFuturesPositions_Success(t *testing.T) {
	m := &mockPort{futuresPositions: []port.FuturesPosition{{Symbol: "BTCUSDT", PositionAmt: "0.1"}}}
	res := callTool(t, m, "get_futures_positions", map[string]any{})
	assertOK(t, res)
	if m.lastFuturesPosSymbol != "" {
		t.Errorf("expected empty symbol filter, got %s", m.lastFuturesPosSymbol)
	}
}

func TestGetFuturesPositions_SymbolFilter(t *testing.T) {
	m := &mockPort{futuresPositions: []port.FuturesPosition{{Symbol: "ETHUSDT", PositionAmt: "2"}}}
	res := callTool(t, m, "get_futures_positions", map[string]any{"symbol": "ETHUSDT"})
	assertOK(t, res)
	if m.lastFuturesPosSymbol != "ETHUSDT" {
		t.Errorf("expected symbol ETHUSDT, got %s", m.lastFuturesPosSymbol)
	}
}

func TestGetFuturesPositions_Error(t *testing.T) {
	m := &mockPort{errFuturesPositions: sentinelErr}
	res := callTool(t, m, "get_futures_positions", map[string]any{})
	assertError(t, res, "get_futures_positions failed")
}

// ─── Account ──────────────────────────────────────────────────────────────────

func TestGetBalance_Success(t *testing.T) {
	m := &mockPort{balances: []port.Balance{{Asset: "BTC", Free: "0.5", Locked: "0.1"}}}
	res := callTool(t, m, "get_balance", map[string]any{})
	assertOK(t, res)
	if !strings.Contains(firstText(res), "BTC") {
		t.Error("expected BTC in balance response")
	}
}

func TestGetBalance_FilteredByAsset(t *testing.T) {
	m := &mockPort{balances: []port.Balance{{Asset: "ETH", Free: "1.0", Locked: "0"}}}
	res := callTool(t, m, "get_balance", map[string]any{"asset": "ETH"})
	assertOK(t, res)
}

func TestGetBalance_Error(t *testing.T) {
	m := &mockPort{errBalance: sentinelErr}
	res := callTool(t, m, "get_balance", map[string]any{})
	assertError(t, res, "get_balance failed")
}

func TestGetFuturesBalance_Success(t *testing.T) {
	m := &mockPort{futuresBalances: []port.FuturesBalance{{Asset: "USDT", Balance: "100.5", AvailableBalance: "95.0"}}}
	res := callTool(t, m, "get_futures_balance", map[string]any{})
	assertOK(t, res)
	if !strings.Contains(firstText(res), "USDT") {
		t.Error("expected USDT in futures balance response")
	}
	if m.lastFuturesBalAsset != "" {
		t.Errorf("expected empty asset filter, got %s", m.lastFuturesBalAsset)
	}
}

func TestGetFuturesBalance_FilteredByAsset(t *testing.T) {
	m := &mockPort{futuresBalances: []port.FuturesBalance{{Asset: "USDT", Balance: "100.5"}}}
	res := callTool(t, m, "get_futures_balance", map[string]any{"asset": "USDT"})
	assertOK(t, res)
	if m.lastFuturesBalAsset != "USDT" {
		t.Errorf("expected asset USDT, got %s", m.lastFuturesBalAsset)
	}
}

func TestGetFuturesBalance_Error(t *testing.T) {
	m := &mockPort{errFuturesBalance: sentinelErr}
	res := callTool(t, m, "get_futures_balance", map[string]any{})
	assertError(t, res, "get_futures_balance failed")
}

func TestGetPositions_Success(t *testing.T) {
	m := &mockPort{futuresPositions: []port.FuturesPosition{{Symbol: "ETHUSDT", PositionAmt: "2"}}}
	res := callTool(t, m, "get_positions", map[string]any{})
	assertOK(t, res)
}

func TestGetPositions_Error(t *testing.T) {
	m := &mockPort{errPositions: sentinelErr}
	res := callTool(t, m, "get_positions", map[string]any{})
	assertError(t, res, "get_positions failed")
}

func TestSetLeverage_Success(t *testing.T) {
	m := &mockPort{}
	res := callTool(t, m, "set_leverage", map[string]any{"symbol": "BTCUSDT", "leverage": float64(10)})
	assertOK(t, res)
	if m.lastSetLeverage.Leverage != 10 {
		t.Errorf("expected leverage 10, got %d", m.lastSetLeverage.Leverage)
	}
}

func TestSetLeverage_Error(t *testing.T) {
	m := &mockPort{errSetLeverage: sentinelErr}
	res := callTool(t, m, "set_leverage", map[string]any{"symbol": "BTCUSDT", "leverage": float64(10)})
	assertError(t, res, "set_leverage failed")
}

func TestSetMarginMode_Success(t *testing.T) {
	m := &mockPort{}
	res := callTool(t, m, "set_margin_mode", map[string]any{"symbol": "BTCUSDT", "marginMode": "ISOLATED"})
	assertOK(t, res)
	if m.lastSetMarginMode.MarginMode != "ISOLATED" {
		t.Errorf("expected ISOLATED, got %s", m.lastSetMarginMode.MarginMode)
	}
}

func TestSetMarginMode_Error(t *testing.T) {
	m := &mockPort{errSetMarginMode: sentinelErr}
	res := callTool(t, m, "set_margin_mode", map[string]any{"symbol": "BTCUSDT", "marginMode": "CROSSED"})
	assertError(t, res, "set_margin_mode failed")
}

func TestTransferFunds_Success(t *testing.T) {
	m := &mockPort{}
	res := callTool(t, m, "transfer_funds", map[string]any{
		"asset": "USDT", "amount": 100.0, "fromAccount": "SPOT", "toAccount": "FUTURES",
	})
	assertOK(t, res)
	if m.lastTransferFunds.Asset != "USDT" {
		t.Errorf("expected USDT, got %s", m.lastTransferFunds.Asset)
	}
}

func TestTransferFunds_Error(t *testing.T) {
	m := &mockPort{errTransferFunds: sentinelErr}
	res := callTool(t, m, "transfer_funds", map[string]any{
		"asset": "USDT", "amount": 100.0, "fromAccount": "SPOT", "toAccount": "FUTURES",
	})
	assertError(t, res, "transfer_funds failed")
}

// ─── System ───────────────────────────────────────────────────────────────────

func TestGetServerInfo_Success(t *testing.T) {
	m := &mockPort{serverInfo: &port.ServerInfo{ServerTime: 1700000000, Version: "1.0.0", Environment: "production"}}
	res := callTool(t, m, "get_server_info", map[string]any{})
	assertOK(t, res)
	if !strings.Contains(firstText(res), "production") {
		t.Error("expected 'production' in server info response")
	}
}

func TestGetServerInfo_Error(t *testing.T) {
	m := &mockPort{errServerInfo: sentinelErr}
	res := callTool(t, m, "get_server_info", map[string]any{})
	assertError(t, res, "get_server_info failed")
}
