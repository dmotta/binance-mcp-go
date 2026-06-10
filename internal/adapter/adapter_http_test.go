package adapter

// Wire-level tests for BUG-001 and BUG-002 (specs/binance-mcp/bugfix.md).
// The Binance HTTP client is mocked with a capturing RoundTripper, so tests
// assert on the real endpoint path and form params without touching the
// network. Against the original code (commit 0ab1db4) the BUG-001 assertions
// fail (trailingDelta was sent as the raw "1.5" string) and the BUG-002
// futures assertions fail (all four operations hit /api/v3/*).

import (
	"context"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"testing"

	binance "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"

	"binance-mcp-go/internal/port"
)

// captureTransport records every request and answers with a canned JSON body.
type captureTransport struct {
	requests []*http.Request
	params   []url.Values // merged query + form params per request
	resp     string
}

func (c *captureTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	merged := r.URL.Query()
	if r.Body != nil {
		b, _ := io.ReadAll(r.Body)
		if vals, err := url.ParseQuery(string(b)); err == nil {
			for k, vs := range vals {
				for _, v := range vs {
					merged.Add(k, v)
				}
			}
		}
	}
	c.requests = append(c.requests, r)
	c.params = append(c.params, merged)
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(c.resp)),
		Header:     make(http.Header),
	}, nil
}

// lastRequest returns the only captured request, failing if count differs.
func (c *captureTransport) lastRequest(t *testing.T) (*http.Request, url.Values) {
	t.Helper()
	if len(c.requests) != 1 {
		t.Fatalf("expected exactly 1 HTTP request, got %d", len(c.requests))
	}
	return c.requests[0], c.params[0]
}

// newWireAdapter builds an adapter whose spot and futures clients share a
// capturing mock transport that replies with resp.
func newWireAdapter(resp string) (*BinanceAdapter, *captureTransport) {
	ct := &captureTransport{resp: resp}
	hc := &http.Client{Transport: ct}
	spot := binance.NewClient("test-key", "test-secret")
	spot.HTTPClient = hc
	fut := futures.NewClient("test-key", "test-secret")
	fut.HTTPClient = hc
	return New(spot, fut, nil, "testnet"), ct
}

const trailingDeltaPattern = `^[0-9]{1,20}$` // Binance -1100 regex for trailingDelta

// ─── BUG-001: callbackRate (percent) → trailingDelta (integer BIPS) ──────────

// Evidence table from the spec: valid percents and the exact BIPS that must
// reach the API. Fails against 0ab1db4, which sent the raw float string.
func TestCreateTrailingStopOrder_BipsConversion(t *testing.T) {
	re := regexp.MustCompile(trailingDeltaPattern)
	cases := []struct {
		callbackRate float64
		wantBips     string
	}{
		{0.1, "10"}, // lower bound
		{0.5, "50"},
		{1.0, "100"},
		{1.5, "150"}, // spec's primary repro case (-1100 on original code)
		{5, "500"},
		{10, "1000"},
		{20, "2000"}, // upper bound (TRAILING_DELTA filter max)
	}
	for _, c := range cases {
		t.Run(strconv.FormatFloat(c.callbackRate, 'g', -1, 64), func(t *testing.T) {
			a, ct := newWireAdapter(`{"symbol":"BTCUSDT","orderId":13,"status":"NEW"}`)
			res, err := a.CreateTrailingStopOrder(context.Background(), port.TrailingStopOrderParams{
				Symbol: "BTCUSDT", Side: "SELL", Quantity: "0.1", CallbackRate: c.callbackRate,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			req, params := ct.lastRequest(t)
			if req.URL.Path != "/api/v3/order" {
				t.Errorf("expected spot path /api/v3/order, got %s", req.URL.Path)
			}
			if got := params.Get("trailingDelta"); got != c.wantBips {
				t.Errorf("callbackRate=%g: expected trailingDelta=%s, got %q", c.callbackRate, c.wantBips, got)
			}
			if got := params.Get("trailingDelta"); !re.MatchString(got) {
				t.Errorf("trailingDelta %q does not match Binance regex %s", got, trailingDeltaPattern)
			}
			if got := params.Get("type"); got != "STOP_LOSS" {
				t.Errorf("expected order type STOP_LOSS, got %q", got)
			}
			if res.OrderID != 13 {
				t.Errorf("expected orderId 13, got %d", res.OrderID)
			}
		})
	}
}

// Out-of-range values from the spec's evidence table must be rejected with a
// clear validation message BEFORE any HTTP request is emitted.
func TestCreateTrailingStopOrder_RejectsOutOfRange(t *testing.T) {
	cases := []float64{21, 150, 0, -1, 0.05}
	for _, rate := range cases {
		t.Run(strconv.FormatFloat(rate, 'g', -1, 64), func(t *testing.T) {
			a, ct := newWireAdapter(`{}`)
			_, err := a.CreateTrailingStopOrder(context.Background(), port.TrailingStopOrderParams{
				Symbol: "BTCUSDT", Side: "SELL", Quantity: "0.1", CallbackRate: rate,
			})
			if err == nil {
				t.Fatalf("callbackRate=%g: expected validation error, got nil", rate)
			}
			if !strings.Contains(err.Error(), "callbackRate must be between 0.1 and 20 (percent)") {
				t.Errorf("callbackRate=%g: unexpected error message: %v", rate, err)
			}
			if len(ct.requests) != 0 {
				t.Errorf("callbackRate=%g: expected no HTTP request, got %d", rate, len(ct.requests))
			}
		})
	}
}

// Property: for any callbackRate in the valid range [0.1, 20], the wire value
// is round(rate*100), an integer in [10, 2000] matching Binance's regex.
func TestCreateTrailingStopOrder_Property(t *testing.T) {
	re := regexp.MustCompile(trailingDeltaPattern)
	rng := rand.New(rand.NewSource(42)) // deterministic
	for i := 0; i < 500; i++ {
		rate := 0.1 + rng.Float64()*19.9
		a, ct := newWireAdapter(`{"symbol":"BTCUSDT","orderId":1,"status":"NEW"}`)
		_, err := a.CreateTrailingStopOrder(context.Background(), port.TrailingStopOrderParams{
			Symbol: "BTCUSDT", Side: "SELL", Quantity: "0.1", CallbackRate: rate,
		})
		if err != nil {
			t.Fatalf("callbackRate=%v: unexpected error: %v", rate, err)
		}
		_, params := ct.lastRequest(t)
		got := params.Get("trailingDelta")
		if !re.MatchString(got) {
			t.Fatalf("callbackRate=%v: trailingDelta %q fails regex", rate, got)
		}
		bips, err := strconv.Atoi(got)
		if err != nil {
			t.Fatalf("callbackRate=%v: trailingDelta %q is not an integer", rate, got)
		}
		if bips < 10 || bips > 2000 {
			t.Fatalf("callbackRate=%v: trailingDelta %d outside [10, 2000]", rate, bips)
		}
		if want := int(math.Round(rate * 100)); bips != want {
			t.Fatalf("callbackRate=%v: expected %d BIPS, got %d", rate, want, bips)
		}
	}
}

// ─── BUG-002: order management must route by market ──────────────────────────

const orderJSON = `{"symbol":"BTCUSDT","orderId":42,"status":"CANCELED","side":"BUY","type":"LIMIT","price":"50000","origQty":"0.001"}`

func TestCancelOrder_RoutesByMarket(t *testing.T) {
	cases := []struct {
		name     string
		market   string
		wantPath string
	}{
		{"default is spot", "", "/api/v3/order"},
		{"explicit spot", "spot", "/api/v3/order"},
		{"futures", "futures", "/fapi/v1/order"}, // fails against 0ab1db4
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ct := newWireAdapter(orderJSON)
			res, err := a.CancelOrder(context.Background(), port.CancelOrderParams{
				Symbol: "BTCUSDT", OrderID: 42, Market: c.market,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			req, params := ct.lastRequest(t)
			if req.URL.Path != c.wantPath {
				t.Errorf("expected path %s, got %s", c.wantPath, req.URL.Path)
			}
			if req.Method != http.MethodDelete {
				t.Errorf("expected DELETE, got %s", req.Method)
			}
			if got := params.Get("orderId"); got != "42" {
				t.Errorf("expected orderId=42, got %q", got)
			}
			if res.Status != "CANCELED" {
				t.Errorf("expected status CANCELED, got %s", res.Status)
			}
		})
	}
}

func TestGetOrderStatus_RoutesByMarket(t *testing.T) {
	cases := []struct {
		name     string
		market   string
		wantPath string
	}{
		{"default is spot", "", "/api/v3/order"},
		{"futures", "futures", "/fapi/v1/order"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ct := newWireAdapter(orderJSON)
			res, err := a.GetOrderStatus(context.Background(), port.GetOrderStatusParams{
				Symbol: "BTCUSDT", OrderID: 42, Market: c.market,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			req, params := ct.lastRequest(t)
			if req.URL.Path != c.wantPath {
				t.Errorf("expected path %s, got %s", c.wantPath, req.URL.Path)
			}
			if req.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", req.Method)
			}
			if got := params.Get("orderId"); got != "42" {
				t.Errorf("expected orderId=42, got %q", got)
			}
			if res.OrderID != 42 {
				t.Errorf("expected orderId 42, got %d", res.OrderID)
			}
		})
	}
}

func TestGetOpenOrders_RoutesByMarket(t *testing.T) {
	cases := []struct {
		name     string
		market   string
		wantPath string
	}{
		{"default is spot", "", "/api/v3/openOrders"},
		{"futures", "futures", "/fapi/v1/openOrders"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a, ct := newWireAdapter(`[` + orderJSON + `]`)
			res, err := a.GetOpenOrders(context.Background(), port.GetOpenOrdersParams{
				Symbol: "BTCUSDT", Market: c.market,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			req, params := ct.lastRequest(t)
			if req.URL.Path != c.wantPath {
				t.Errorf("expected path %s, got %s", c.wantPath, req.URL.Path)
			}
			if got := params.Get("symbol"); got != "BTCUSDT" {
				t.Errorf("expected symbol=BTCUSDT, got %q", got)
			}
			if len(res) != 1 || res[0].OrderID != 42 {
				t.Errorf("expected one order with id 42, got %+v", res)
			}
		})
	}
}

func TestCancelAllOrders_RoutesByMarket(t *testing.T) {
	t.Run("default is spot", func(t *testing.T) {
		// orderListId:-1 marks a non-OCO order in the SDK's response parser.
		a, ct := newWireAdapter(`[{"symbol":"BTCUSDT","orderId":42,"orderListId":-1,"status":"CANCELED","side":"BUY","type":"LIMIT","price":"50000","origQty":"0.001"}]`)
		res, err := a.CancelAllOrders(context.Background(), port.CancelAllOrdersParams{Symbol: "BTCUSDT"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, params := ct.lastRequest(t)
		if req.URL.Path != "/api/v3/openOrders" {
			t.Errorf("expected path /api/v3/openOrders, got %s", req.URL.Path)
		}
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if got := params.Get("symbol"); got != "BTCUSDT" {
			t.Errorf("expected symbol=BTCUSDT, got %q", got)
		}
		if len(res) != 1 || res[0].Status != "CANCELED" {
			t.Errorf("expected one CANCELED order, got %+v", res)
		}
	})
	t.Run("futures", func(t *testing.T) {
		a, ct := newWireAdapter(`{"code":200,"msg":"The operation of cancel all open order is done."}`)
		res, err := a.CancelAllOrders(context.Background(), port.CancelAllOrdersParams{
			Symbol: "BTCUSDT", Market: "futures",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		req, params := ct.lastRequest(t)
		if req.URL.Path != "/fapi/v1/allOpenOrders" {
			t.Errorf("expected path /fapi/v1/allOpenOrders, got %s", req.URL.Path)
		}
		if req.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", req.Method)
		}
		if got := params.Get("symbol"); got != "BTCUSDT" {
			t.Errorf("expected symbol=BTCUSDT, got %q", got)
		}
		if len(res) != 1 || res[0].Status != "CANCELED" {
			t.Errorf("expected synthetic CANCELED ack, got %+v", res)
		}
	})
}
