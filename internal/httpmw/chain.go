package httpmw

import "net/http"

type Middleware func(http.RoundTripper) http.RoundTripper

// Chain wraps base with each middleware outermost-first.
// Chain(base, A, B, C) → A(B(C(base)))
func Chain(base http.RoundTripper, mws ...Middleware) http.RoundTripper {
	rt := base
	for i := len(mws) - 1; i >= 0; i-- {
		rt = mws[i](rt)
	}
	return rt
}
