package httpmw

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// signedURL mirrors the shape go-binance produces for a signed order: all
// parameters plus the HMAC signature live in the query string.
const signedURL = "https://api.binance.com/api/v3/order?symbol=BTCUSDT&side=BUY&timestamp=1700000000000&signature=8fa1c0deadbeef99"

const signatureValue = "8fa1c0deadbeef99"

// newRecordingTracer installs an in-memory exporter and returns it plus a
// cleanup that restores the previous global tracer provider.
func newRecordingTracer(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() {
		_ = tp.Shutdown(t.Context())
		otel.SetTracerProvider(prev)
	})
	return exp
}

// TestOtel_RedactsSignatureFromSpanURL is the logging-leak guarantee: the
// exported span must never carry the request signature, because spans are
// written to the on-disk log file. OWASP A09:2021.
func TestOtel_RedactsSignatureFromSpanURL(t *testing.T) {
	exp := newRecordingTracer(t)

	f := &fakeRT{responses: []*http.Response{resp(200)}}
	rt := NewOtelTransport(f)
	r, _ := http.NewRequest(http.MethodPost, signedURL, nil)

	if _, err := rt.RoundTrip(r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}

	var urlAttr string
	var found bool
	for _, a := range spans[0].Attributes {
		if string(a.Key) == "http.url" {
			urlAttr = a.Value.AsString()
			found = true
		}
	}
	if !found {
		t.Fatal("span is missing the http.url attribute")
	}
	if strings.Contains(urlAttr, signatureValue) {
		t.Fatalf("signature leaked into span attribute: %q", urlAttr)
	}
	if !strings.Contains(urlAttr, "symbol=BTCUSDT") {
		t.Fatalf("redaction destroyed useful trace context: %q", urlAttr)
	}
}

// TestOtel_RedactsSignatureFromRecordedError covers the secondary vector: a
// transport error whose message embeds the signed URL must be redacted before
// it is recorded on the span, while the caller still receives the original.
func TestOtel_RedactsSignatureFromRecordedError(t *testing.T) {
	exp := newRecordingTracer(t)

	transportErr := errors.New(`Post "` + signedURL + `": dial tcp: i/o timeout`)
	f := &fakeRT{
		responses: []*http.Response{nil},
		errs:      []error{transportErr},
	}
	rt := NewOtelTransport(f)
	r, _ := http.NewRequest(http.MethodPost, signedURL, nil)

	_, err := rt.RoundTrip(r)
	if err == nil {
		t.Fatal("expected the transport error to surface to the caller")
	}
	// The caller must still see the untouched original error.
	if !errors.Is(err, transportErr) {
		t.Fatalf("caller should receive the original error, got %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("want 1 span, got %d", len(spans))
	}
	if got := spans[0].Status.Description; strings.Contains(got, signatureValue) {
		t.Fatalf("signature leaked into span status: %q", got)
	}
	for _, ev := range spans[0].Events {
		for _, a := range ev.Attributes {
			if strings.Contains(a.Value.AsString(), signatureValue) {
				t.Fatalf("signature leaked into span event %q: %q", ev.Name, a.Value.AsString())
			}
		}
	}
}
