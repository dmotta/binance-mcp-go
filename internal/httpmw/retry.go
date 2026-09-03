package httpmw

import (
	"net/http"
	"time"

	"github.com/jpillora/backoff"
)

const maxRetryAttempts = 3

type retryTransport struct {
	next http.RoundTripper
}

func NewRetryTransport(next http.RoundTripper) http.RoundTripper {
	return &retryTransport{next: next}
}

// newBackoff returns a fresh backoff per call so concurrent requests never
// share mutable timing state.
func newBackoff() *backoff.Backoff {
	return &backoff.Backoff{
		Min:    200 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2,
		Jitter: true,
	}
}

// isIdempotent reports whether an HTTP method is safe to send more than once.
// Only GET and HEAD qualify: they have no side effects, so a retry after a 5xx
// or transport error cannot change account state.
//
// POST/DELETE/PUT/PATCH are NOT retried. Binance places order parameters and the
// request signature in the query string with an empty body, so a body-rewind
// heuristic cannot distinguish a trade from a read. More importantly, a 5xx or a
// dropped connection may arrive AFTER Binance has already executed the request;
// replaying it would duplicate an order, cancellation, or fund transfer. This is
// a fail-safe default (NIST SP 800-160 secure-design): when the outcome of a
// state-changing request is unknown, do not repeat it. (OWASP A04:2021.)
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead:
		return true
	default:
		return false
	}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Non-idempotent requests are sent at most once. See isIdempotent.
	if !isIdempotent(req.Method) {
		return t.next.RoundTrip(req)
	}

	b := newBackoff()
	var (
		resp *http.Response
		err  error
	)
	for attempt := 0; attempt < maxRetryAttempts; attempt++ {
		attemptReq := req
		if attempt > 0 && req.GetBody != nil {
			body, gErr := req.GetBody()
			if gErr != nil {
				return resp, err
			}
			clone := req.Clone(req.Context())
			clone.Body = body
			attemptReq = clone
		}

		resp, err = t.next.RoundTrip(attemptReq)
		if err != nil {
			if attempt < maxRetryAttempts-1 {
				time.Sleep(b.Duration())
				continue
			}
			return resp, err
		}
		if resp.StatusCode >= 500 && attempt < maxRetryAttempts-1 {
			resp.Body.Close()
			time.Sleep(b.Duration())
			continue
		}
		return resp, nil
	}
	return resp, err
}
