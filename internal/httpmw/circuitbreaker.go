package httpmw

import (
	"fmt"
	"net/http"

	"github.com/sony/gobreaker/v2"
)

type circuitBreakerTransport struct {
	next http.RoundTripper
	cb   *gobreaker.CircuitBreaker[*http.Response]
}

func NewCircuitBreakerTransport(next http.RoundTripper) http.RoundTripper {
	settings := gobreaker.Settings{
		Name:        "binance",
		MaxRequests: 1,
		Interval:    0,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
	}
	cb := gobreaker.NewCircuitBreaker[*http.Response](settings)
	return &circuitBreakerTransport{next: next, cb: cb}
}

func (t *circuitBreakerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.cb.Execute(func() (*http.Response, error) {
		r, e := t.next.RoundTrip(req)
		if e != nil {
			return nil, e
		}
		if r.StatusCode >= 500 {
			return r, fmt.Errorf("server error: %d", r.StatusCode)
		}
		return r, nil
	})
	if err != nil && resp != nil {
		// CB tripped on a 5xx: return the response so the caller sees the status
		return resp, nil
	}
	return resp, err
}
