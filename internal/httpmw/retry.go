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

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// A request whose body cannot be rewound must not be retried: replaying it
	// would send an empty/partial body (e.g. a signed order with no payload),
	// which Binance rejects or, worse, mis-signs. Trading calls are not
	// idempotent, so we only ever send them once.
	if req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
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
