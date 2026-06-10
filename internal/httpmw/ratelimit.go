package httpmw

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

const (
	weightHeader    = "X-Mbx-Used-Weight-1M"
	weightThreshold = 1100
	weightLimit     = 1200
)

type rateLimitTransport struct {
	next       http.RoundTripper
	usedWeight atomic.Int64
	sleep      func(time.Duration) // injectable for tests; defaults to time.Sleep
}

func NewRateLimitTransport(next http.RoundTripper) http.RoundTripper {
	return &rateLimitTransport{next: next, sleep: time.Sleep}
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if w := t.usedWeight.Load(); w >= weightThreshold {
		now := time.Now()
		sleepUntil := now.Truncate(time.Minute).Add(time.Minute)
		t.sleep(time.Until(sleepUntil))
		t.usedWeight.Store(0)
	}

	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if v := resp.Header.Get(weightHeader); v != "" {
		if w, parseErr := strconv.ParseInt(v, 10, 64); parseErr == nil {
			t.usedWeight.Store(w)
		}
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == 418 {
		retryAfter := resp.Header.Get("Retry-After")
		secs, parseErr := strconv.ParseInt(retryAfter, 10, 64)
		switch {
		case parseErr == nil && secs > 0:
			t.sleep(time.Duration(secs) * time.Second)
		case parseErr == nil && secs == 0:
			// Server explicitly says retry immediately; don't block.
		default:
			t.sleep(time.Minute)
		}
	}

	return resp, nil
}
