package adapter

// Wire-level tests for the futures additions. Conditional orders
// (STOP_MARKET / TAKE_PROFIT_MARKET) must go to the Algo Order API
// (/fapi/v1/algoOrder, algoType=CONDITIONAL): Binance rejects them on
// /fapi/v1/order with -4120 (bugfix002.md). Also covers algo order
// list/cancel, futures wallet balances, futures account trades, and the
// symbol filter on position risk. Same capturing-RoundTripper setup as
// adapter_http_test.go.

import (
	"context"
	"strings"
	"testing"

	"binance-mcp-go/internal/port"
)

// ─── create_contract_order: STOP_MARKET / TAKE_PROFIT_MARKET ─────────────────

func TestCreateContractOrder_StopMarketWire(t *testing.T) {
	a, ct := newWireAdapter(`{"algoId":2146760,"symbol":"BTCUSDT","side":"SELL","algoStatus":"NEW","triggerPrice":"45000"}`)
	res, err := a.CreateContractOrder(context.Background(), port.ContractOrderParams{
		Symbol: "BTCUSDT", Side: "SELL", Type: "STOP_MARKET",
		Quantity: "0.01", StopPrice: "45000", ReduceOnly: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, params := ct.lastRequest(t)
	// -4120: conditional orders are rejected on /fapi/v1/order and must use
	// the Algo Order API.
	if req.URL.Path != "/fapi/v1/algoOrder" {
		t.Errorf("expected path /fapi/v1/algoOrder, got %s", req.URL.Path)
	}
	if got := params.Get("algoType"); got != "CONDITIONAL" {
		t.Errorf("expected algoType=CONDITIONAL, got %q", got)
	}
	if got := params.Get("type"); got != "STOP_MARKET" {
		t.Errorf("expected type STOP_MARKET, got %q", got)
	}
	if got := params.Get("triggerPrice"); got != "45000" {
		t.Errorf("expected triggerPrice=45000, got %q", got)
	}
	if got := params.Get("reduceOnly"); got != "true" {
		t.Errorf("expected reduceOnly=true, got %q", got)
	}
	if got := params.Get("quantity"); got != "0.01" {
		t.Errorf("expected quantity=0.01, got %q", got)
	}
	if res.AlgoID != 2146760 {
		t.Errorf("expected algoId 2146760, got %d", res.AlgoID)
	}
	if res.OrderID != 0 {
		t.Errorf("expected no orderId for algo orders, got %d", res.OrderID)
	}
	if res.Status != "NEW" {
		t.Errorf("expected status NEW, got %s", res.Status)
	}
}

func TestCreateContractOrder_ClosePositionWire(t *testing.T) {
	a, ct := newWireAdapter(`{"algoId":2146761,"symbol":"BTCUSDT","side":"SELL","algoStatus":"NEW","closePosition":true}`)
	_, err := a.CreateContractOrder(context.Background(), port.ContractOrderParams{
		Symbol: "BTCUSDT", Side: "SELL", Type: "TAKE_PROFIT_MARKET",
		StopPrice: "60000", ClosePosition: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, params := ct.lastRequest(t)
	if req.URL.Path != "/fapi/v1/algoOrder" {
		t.Errorf("expected path /fapi/v1/algoOrder, got %s", req.URL.Path)
	}
	if got := params.Get("closePosition"); got != "true" {
		t.Errorf("expected closePosition=true, got %q", got)
	}
	// Binance rejects quantity and reduceOnly together with closePosition.
	if got := params.Get("quantity"); got != "" {
		t.Errorf("expected no quantity param, got %q", got)
	}
	if got := params.Get("reduceOnly"); got != "" {
		t.Errorf("expected no reduceOnly param, got %q", got)
	}
	if got := params.Get("triggerPrice"); got != "60000" {
		t.Errorf("expected triggerPrice=60000, got %q", got)
	}
}

// Regular (non-conditional) futures orders must keep using /fapi/v1/order.
func TestCreateContractOrder_MarketStillRegularEndpoint(t *testing.T) {
	a, ct := newWireAdapter(`{"symbol":"BTCUSDT","orderId":77,"status":"NEW"}`)
	res, err := a.CreateContractOrder(context.Background(), port.ContractOrderParams{
		Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET", Quantity: "0.002",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, params := ct.lastRequest(t)
	if req.URL.Path != "/fapi/v1/order" {
		t.Errorf("expected path /fapi/v1/order, got %s", req.URL.Path)
	}
	if got := params.Get("type"); got != "MARKET" {
		t.Errorf("expected type MARKET, got %q", got)
	}
	if res.OrderID != 77 {
		t.Errorf("expected orderId 77, got %d", res.OrderID)
	}
}

// ─── algo orders: list and cancel ────────────────────────────────────────────

func TestGetOpenAlgoOrders_Wire(t *testing.T) {
	a, ct := newWireAdapter(`[{"algoId":2146760,"symbol":"BTCUSDT","side":"SELL","orderType":"STOP_MARKET","algoStatus":"NEW","triggerPrice":"45000","quantity":"0.002","reduceOnly":true,"closePosition":false}]`)
	res, err := a.GetOpenAlgoOrders(context.Background(), "BTCUSDT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, params := ct.lastRequest(t)
	if req.URL.Path != "/fapi/v1/openAlgoOrders" {
		t.Errorf("expected path /fapi/v1/openAlgoOrders, got %s", req.URL.Path)
	}
	if got := params.Get("symbol"); got != "BTCUSDT" {
		t.Errorf("expected symbol=BTCUSDT, got %q", got)
	}
	if len(res) != 1 {
		t.Fatalf("expected 1 algo order, got %d", len(res))
	}
	o := res[0]
	if o.AlgoID != 2146760 || o.Type != "STOP_MARKET" || o.TriggerPrice != "45000" || !o.ReduceOnly {
		t.Errorf("unexpected mapped algo order: %+v", o)
	}
}

func TestGetOpenAlgoOrders_NoSymbol(t *testing.T) {
	a, ct := newWireAdapter(`[]`)
	res, err := a.GetOpenAlgoOrders(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, params := ct.lastRequest(t)
	if got := params.Get("symbol"); got != "" {
		t.Errorf("expected no symbol param, got %q", got)
	}
	if len(res) != 0 {
		t.Errorf("expected empty list, got %+v", res)
	}
}

func TestCancelAlgoOrder_Wire(t *testing.T) {
	a, ct := newWireAdapter(`{"algoId":2146760,"clientAlgoId":"abc","code":"200","msg":"success"}`)
	res, err := a.CancelAlgoOrder(context.Background(), 2146760)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, params := ct.lastRequest(t)
	if req.URL.Path != "/fapi/v1/algoOrder" {
		t.Errorf("expected path /fapi/v1/algoOrder, got %s", req.URL.Path)
	}
	if req.Method != "DELETE" {
		t.Errorf("expected DELETE, got %s", req.Method)
	}
	if got := params.Get("algoId"); got != "2146760" {
		t.Errorf("expected algoId=2146760, got %q", got)
	}
	if res.AlgoID != 2146760 {
		t.Errorf("expected algoId 2146760 in result, got %d", res.AlgoID)
	}
}

func TestCancelAlgoOrder_RejectsInvalidID(t *testing.T) {
	for _, id := range []int64{0, -5} {
		a, ct := newWireAdapter(`{}`)
		_, err := a.CancelAlgoOrder(context.Background(), id)
		if err == nil {
			t.Fatalf("algoId=%d: expected validation error, got nil", id)
		}
		if !strings.Contains(err.Error(), "algoId must be a positive integer") {
			t.Errorf("algoId=%d: unexpected error: %v", id, err)
		}
		if len(ct.requests) != 0 {
			t.Errorf("algoId=%d: expected no HTTP request, got %d", id, len(ct.requests))
		}
	}
}

func TestCreateContractOrder_Validation(t *testing.T) {
	cases := []struct {
		name    string
		params  port.ContractOrderParams
		wantErr string
	}{
		{
			"stop market without stopPrice",
			port.ContractOrderParams{Symbol: "BTCUSDT", Side: "SELL", Type: "STOP_MARKET", Quantity: "0.01"},
			"stopPrice is required",
		},
		{
			"take profit market without stopPrice",
			port.ContractOrderParams{Symbol: "BTCUSDT", Side: "SELL", Type: "TAKE_PROFIT_MARKET", Quantity: "0.01"},
			"stopPrice is required",
		},
		{
			"closePosition on MARKET",
			port.ContractOrderParams{Symbol: "BTCUSDT", Side: "SELL", Type: "MARKET", ClosePosition: true},
			"closePosition is only valid",
		},
		{
			"missing quantity without closePosition",
			port.ContractOrderParams{Symbol: "BTCUSDT", Side: "BUY", Type: "MARKET"},
			"quantity is required",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ct := newWireAdapter(`{}`)
			_, err := a.CreateContractOrder(context.Background(), c.params)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("unexpected error message: %v", err)
			}
			if len(ct.requests) != 0 {
				t.Errorf("expected no HTTP request, got %d", len(ct.requests))
			}
		})
	}
}

// ─── get_my_trades: route by market ──────────────────────────────────────────

func TestGetMyTrades_RoutesByMarket(t *testing.T) {
	t.Run("default is spot", func(t *testing.T) {
		a, ct := newWireAdapter(`[{"id":1,"symbol":"BTCUSDT","price":"50000","qty":"0.001","isBuyer":true,"time":1700000000000}]`)
		res, err := a.GetMyTrades(context.Background(), port.GetMyTradesParams{Symbol: "BTCUSDT", Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, params := ct.lastRequest(t)
		if req.URL.Path != "/api/v3/myTrades" {
			t.Errorf("expected path /api/v3/myTrades, got %s", req.URL.Path)
		}
		if got := params.Get("limit"); got != "10" {
			t.Errorf("expected limit=10, got %q", got)
		}
		if len(res) != 1 || !res[0].IsBuyer {
			t.Errorf("expected one buyer trade, got %+v", res)
		}
	})
	t.Run("futures", func(t *testing.T) {
		a, ct := newWireAdapter(`[{"buyer":false,"commission":"-0.07","id":698759,"orderId":25851813,"price":"7819.01","qty":"0.002","realizedPnl":"-0.91","side":"SELL","positionSide":"SHORT","symbol":"BTCUSDT","time":1569514978020}]`)
		res, err := a.GetMyTrades(context.Background(), port.GetMyTradesParams{
			Symbol: "BTCUSDT", Limit: 10, Market: "futures",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, params := ct.lastRequest(t)
		if req.URL.Path != "/fapi/v1/userTrades" {
			t.Errorf("expected path /fapi/v1/userTrades, got %s", req.URL.Path)
		}
		if got := params.Get("symbol"); got != "BTCUSDT" {
			t.Errorf("expected symbol=BTCUSDT, got %q", got)
		}
		if len(res) != 1 {
			t.Fatalf("expected 1 trade, got %d", len(res))
		}
		if res[0].RealizedPnl != "-0.91" || res[0].Side != "SELL" || res[0].Qty != "0.002" {
			t.Errorf("unexpected mapped trade: %+v", res[0])
		}
	})
}

// ─── get_futures_balance ─────────────────────────────────────────────────────

const futuresBalanceJSON = `[
	{"accountAlias":"x","asset":"USDT","balance":"100.5","crossWalletBalance":"100.5","crossUnPnl":"0.1","availableBalance":"95.0","maxWithdrawAmount":"95.0"},
	{"accountAlias":"x","asset":"BTC","balance":"0.00000000","crossWalletBalance":"0","crossUnPnl":"0","availableBalance":"0","maxWithdrawAmount":"0"}
]`

func TestGetFuturesBalance_Wire(t *testing.T) {
	a, ct := newWireAdapter(futuresBalanceJSON)
	res, err := a.GetFuturesBalance(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, _ := ct.lastRequest(t)
	if req.URL.Path != "/fapi/v3/balance" {
		t.Errorf("expected path /fapi/v3/balance, got %s", req.URL.Path)
	}
	if len(res) != 1 {
		t.Fatalf("expected zero balances filtered out (1 entry), got %d: %+v", len(res), res)
	}
	if res[0].Asset != "USDT" || res[0].Balance != "100.5" || res[0].AvailableBalance != "95.0" {
		t.Errorf("unexpected balance: %+v", res[0])
	}
}

func TestGetFuturesBalance_AssetFilter(t *testing.T) {
	t.Run("matching asset", func(t *testing.T) {
		a, _ := newWireAdapter(futuresBalanceJSON)
		res, err := a.GetFuturesBalance(context.Background(), "BTC")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// An explicitly requested asset is returned even when zero.
		if len(res) != 1 || res[0].Asset != "BTC" {
			t.Errorf("expected only BTC entry, got %+v", res)
		}
	})
	t.Run("unknown asset", func(t *testing.T) {
		a, _ := newWireAdapter(futuresBalanceJSON)
		res, err := a.GetFuturesBalance(context.Background(), "DOGE")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 0 {
			t.Errorf("expected no entries, got %+v", res)
		}
	})
}

// ─── get_futures_positions: symbol filter ────────────────────────────────────

const positionRiskJSON = `[{"symbol":"BTCUSDT","positionAmt":"0.004","entryPrice":"50000","unRealizedProfit":"1.5","liquidationPrice":"40000","leverage":"10","positionSide":"BOTH"}]`

func TestGetFuturesPositions_SymbolParam(t *testing.T) {
	t.Run("with symbol", func(t *testing.T) {
		a, ct := newWireAdapter(positionRiskJSON)
		res, err := a.GetFuturesPositions(context.Background(), "BTCUSDT")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, params := ct.lastRequest(t)
		if req.URL.Path != "/fapi/v2/positionRisk" {
			t.Errorf("expected path /fapi/v2/positionRisk, got %s", req.URL.Path)
		}
		if got := params.Get("symbol"); got != "BTCUSDT" {
			t.Errorf("expected symbol=BTCUSDT, got %q", got)
		}
		if len(res) != 1 || res[0].Symbol != "BTCUSDT" {
			t.Errorf("expected one BTCUSDT position, got %+v", res)
		}
	})
	t.Run("without symbol", func(t *testing.T) {
		a, ct := newWireAdapter(positionRiskJSON)
		_, err := a.GetFuturesPositions(context.Background(), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		_, params := ct.lastRequest(t)
		if got := params.Get("symbol"); got != "" {
			t.Errorf("expected no symbol param, got %q", got)
		}
	})
}
