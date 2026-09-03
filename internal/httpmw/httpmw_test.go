package httpmw

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// fakeRT is a programmable RoundTripper for tests.
type fakeRT struct {
	mu        sync.Mutex
	calls     int32
	bodies    []string // captured request bodies, one per call
	responses []*http.Response
	errs      []error
}

func (f *fakeRT) RoundTrip(req *http.Request) (*http.Response, error) {
	n := atomic.AddInt32(&f.calls, 1) - 1
	body := ""
	if req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		body = string(b)
	}
	f.mu.Lock()
	f.bodies = append(f.bodies, body)
	f.mu.Unlock()

	i := int(n)
	var resp *http.Response
	var err error
	if i < len(f.responses) {
		resp = f.responses[i]
	}
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return resp, err
}

func resp(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

// reqWithBody builds a POST request; used by the circuit-breaker and rate-limit
// tests, whose behavior is independent of the HTTP method.
func reqWithBody(body string) *http.Request {
	r, _ := http.NewRequest(http.MethodPost, "https://api.binance.com/order", strings.NewReader(body))
	return r
}

// reqGet builds an idempotent GET request — the only kind the retry layer is
// allowed to replay.
func reqGet() *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "https://api.binance.com/api/v3/order", nil)
	return r
}

// reqGetWithBody builds a GET carrying a rewindable body, to exercise the
// body-rewind clone path on a retryable method.
func reqGetWithBody(body string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "https://api.binance.com/api/v3/order", strings.NewReader(body))
	return r
}

// reqPost builds a non-idempotent POST — an order-placement shape that must
// never be retried, even when its body is rewindable.
func reqPost(body string) *http.Request {
	r, _ := http.NewRequest(http.MethodPost, "https://api.binance.com/api/v3/order", strings.NewReader(body))
	return r
}

// ─── Retry ───────────────────────────────────────────────────────────────────

func TestRetry_SuccessNoRetry(t *testing.T) {
	f := &fakeRT{responses: []*http.Response{resp(200)}}
	rt := NewRetryTransport(f)
	r, err := rt.RoundTrip(reqGet())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.StatusCode != 200 {
		t.Fatalf("want 200, got %d", r.StatusCode)
	}
	if f.calls != 1 {
		t.Fatalf("want 1 call, got %d", f.calls)
	}
}

func TestRetry_RetriesOn5xxThenSucceeds(t *testing.T) {
	f := &fakeRT{responses: []*http.Response{resp(503), resp(200)}}
	rt := NewRetryTransport(f)
	r, err := rt.RoundTrip(reqGet())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.StatusCode != 200 {
		t.Fatalf("want 200, got %d", r.StatusCode)
	}
	if f.calls != 2 {
		t.Fatalf("want 2 calls, got %d", f.calls)
	}
}

func TestRetry_RewindsBodyOnEachAttempt(t *testing.T) {
	f := &fakeRT{responses: []*http.Response{resp(500), resp(500), resp(200)}}
	rt := NewRetryTransport(f)
	if _, err := rt.RoundTrip(reqGetWithBody("query-payload")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.calls != 3 {
		t.Fatalf("want 3 calls, got %d", f.calls)
	}
	for i, b := range f.bodies {
		if b != "query-payload" {
			t.Fatalf("attempt %d sent body %q, want full body each time", i, b)
		}
	}
}

func TestRetry_RetriesOnNetworkError(t *testing.T) {
	f := &fakeRT{
		responses: []*http.Response{nil, resp(200)},
		errs:      []error{errors.New("conn reset"), nil},
	}
	rt := NewRetryTransport(f)
	r, err := rt.RoundTrip(reqGet())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil || r.StatusCode != 200 {
		t.Fatalf("want 200 after retry")
	}
	if f.calls != 2 {
		t.Fatalf("want 2 calls, got %d", f.calls)
	}
}

func TestRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	f := &fakeRT{
		responses: []*http.Response{nil, nil, nil},
		errs:      []error{errors.New("e"), errors.New("e"), errors.New("e")},
	}
	rt := NewRetryTransport(f)
	_, err := rt.RoundTrip(reqGet())
	if err == nil {
		t.Fatal("expected error after exhausting attempts")
	}
	if f.calls != maxRetryAttempts {
		t.Fatalf("want %d calls, got %d", maxRetryAttempts, f.calls)
	}
}

// TestRetry_PostNotRetriedOn5xx is the core safety guarantee: a state-changing
// POST (order placement) that returns 5xx must be sent exactly once. Binance may
// have executed the order before returning the error, so a retry could duplicate
// the trade. OWASP A04:2021.
func TestRetry_PostNotRetriedOn5xx(t *testing.T) {
	f := &fakeRT{responses: []*http.Response{resp(500), resp(200)}}
	rt := NewRetryTransport(f)

	out, err := rt.RoundTrip(reqPost("symbol=BTCUSDT&side=BUY&signature=abc"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.StatusCode != 500 {
		t.Fatalf("want passthrough 500, got %d", out.StatusCode)
	}
	if f.calls != 1 {
		t.Fatalf("non-idempotent POST must be sent once, got %d calls", f.calls)
	}
}

// TestRetry_PostNotRetriedOnNetworkError: a dropped connection on a POST must
// not be retried either — the request may already have reached Binance.
func TestRetry_PostNotRetriedOnNetworkError(t *testing.T) {
	f := &fakeRT{
		responses: []*http.Response{nil, resp(200)},
		errs:      []error{errors.New("conn reset"), nil},
	}
	rt := NewRetryTransport(f)

	_, err := rt.RoundTrip(reqPost("x"))
	if err == nil {
		t.Fatal("expected the transport error to surface without a retry")
	}
	if f.calls != 1 {
		t.Fatalf("non-idempotent POST must be sent once, got %d calls", f.calls)
	}
}

// TestRetry_DeleteNotRetried: order cancellation (DELETE) is likewise sent once.
func TestRetry_DeleteNotRetried(t *testing.T) {
	f := &fakeRT{responses: []*http.Response{resp(503), resp(200)}}
	rt := NewRetryTransport(f)
	r, _ := http.NewRequest(http.MethodDelete, "https://api.binance.com/api/v3/order", nil)

	out, err := rt.RoundTrip(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.StatusCode != 503 {
		t.Fatalf("want passthrough 503, got %d", out.StatusCode)
	}
	if f.calls != 1 {
		t.Fatalf("non-idempotent DELETE must be sent once, got %d calls", f.calls)
	}
}

// ─── Circuit Breaker ─────────────────────────────────────────────────────────

func TestCircuitBreaker_PassesThroughSuccess(t *testing.T) {
	f := &fakeRT{responses: []*http.Response{resp(200)}}
	rt := NewCircuitBreakerTransport(f)
	r, err := rt.RoundTrip(reqWithBody(""))
	if err != nil || r.StatusCode != 200 {
		t.Fatalf("want 200 no error, got %v %v", r, err)
	}
}

func TestCircuitBreaker_TripsAfterConsecutiveFailures(t *testing.T) {
	responses := make([]*http.Response, 10)
	for i := range responses {
		responses[i] = resp(500)
	}
	f := &fakeRT{responses: responses}
	rt := NewCircuitBreakerTransport(f)

	// 5 consecutive 5xx should trip the breaker.
	for i := 0; i < 5; i++ {
		rt.RoundTrip(reqWithBody(""))
	}
	before := f.calls
	// Next call should be short-circuited (breaker open) → no new downstream call.
	_, err := rt.RoundTrip(reqWithBody(""))
	if err == nil {
		t.Fatal("expected breaker-open error")
	}
	if f.calls != before {
		t.Fatalf("breaker open should not call downstream: before=%d after=%d", before, f.calls)
	}
}

func TestCircuitBreaker_Returns5xxResponseToCaller(t *testing.T) {
	f := &fakeRT{responses: []*http.Response{resp(502)}}
	rt := NewCircuitBreakerTransport(f)
	r, err := rt.RoundTrip(reqWithBody(""))
	if err != nil {
		t.Fatalf("first 5xx should surface the response, not an error: %v", err)
	}
	if r == nil || r.StatusCode != 502 {
		t.Fatalf("want 502 response, got %v", r)
	}
}

// ─── Rate Limit ──────────────────────────────────────────────────────────────

func TestRateLimit_PassesThroughAndTracksWeight(t *testing.T) {
	r200 := resp(200)
	r200.Header.Set(weightHeader, "42")
	f := &fakeRT{responses: []*http.Response{r200}}
	rt := NewRateLimitTransport(f).(*rateLimitTransport)

	out, err := rt.RoundTrip(reqWithBody(""))
	if err != nil || out.StatusCode != 200 {
		t.Fatalf("want 200, got %v %v", out, err)
	}
	if got := rt.usedWeight.Load(); got != 42 {
		t.Fatalf("want tracked weight 42, got %d", got)
	}
}

func TestRateLimit_PropagatesError(t *testing.T) {
	f := &fakeRT{responses: []*http.Response{nil}, errs: []error{errors.New("boom")}}
	rt := NewRateLimitTransport(f)
	if _, err := rt.RoundTrip(reqWithBody("")); err == nil {
		t.Fatal("expected error to propagate")
	}
}

func TestRateLimit_Handles429WithRetryAfter(t *testing.T) {
	r := resp(http.StatusTooManyRequests)
	r.Header.Set("Retry-After", "5")
	f := &fakeRT{responses: []*http.Response{r}}
	rt := NewRateLimitTransport(f).(*rateLimitTransport)

	var slept time.Duration
	rt.sleep = func(d time.Duration) { slept = d } // no real blocking

	out, err := rt.RoundTrip(reqWithBody(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("want 429 surfaced, got %d", out.StatusCode)
	}
	if slept != 5*time.Second {
		t.Fatalf("want sleep 5s from Retry-After, got %v", slept)
	}
}

func TestRateLimit_429ZeroRetryAfterDoesNotBlock(t *testing.T) {
	r := resp(http.StatusTooManyRequests)
	r.Header.Set("Retry-After", "0")
	f := &fakeRT{responses: []*http.Response{r}}
	rt := NewRateLimitTransport(f).(*rateLimitTransport)

	called := false
	rt.sleep = func(time.Duration) { called = true }

	if _, err := rt.RoundTrip(reqWithBody("")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("Retry-After: 0 must not sleep")
	}
}

func TestRateLimit_ThresholdTriggersSleep(t *testing.T) {
	f := &fakeRT{responses: []*http.Response{resp(200)}}
	rt := NewRateLimitTransport(f).(*rateLimitTransport)
	rt.usedWeight.Store(weightThreshold) // at/over threshold

	slept := false
	rt.sleep = func(time.Duration) { slept = true }

	if _, err := rt.RoundTrip(reqWithBody("")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slept {
		t.Fatal("expected sleep when weight at threshold")
	}
	if rt.usedWeight.Load() != 0 {
		t.Fatalf("weight should reset to 0 after sleeping, got %d", rt.usedWeight.Load())
	}
}

// ─── Chain ───────────────────────────────────────────────────────────────────

func TestChain_OrdersOutermostFirst(t *testing.T) {
	var order []string
	mw := func(tag string) Middleware {
		return func(next http.RoundTripper) http.RoundTripper {
			return rtFunc(func(req *http.Request) (*http.Response, error) {
				order = append(order, tag)
				return next.RoundTrip(req)
			})
		}
	}
	base := rtFunc(func(req *http.Request) (*http.Response, error) {
		order = append(order, "base")
		return resp(200), nil
	})
	rt := Chain(base, mw("A"), mw("B"), mw("C"))
	rt.RoundTrip(reqWithBody(""))

	got := strings.Join(order, ",")
	if got != "A,B,C,base" {
		t.Fatalf("want A,B,C,base got %s", got)
	}
}

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
