package httpmw

import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type otelTransport struct {
	next    http.RoundTripper
	tracer  trace.Tracer
	counter metric.Int64Counter
}

func NewOtelTransport(next http.RoundTripper) http.RoundTripper {
	tracer := otel.Tracer("binance-mcp/http")
	meter := otel.Meter("binance-mcp/http")
	counter, _ := meter.Int64Counter("http.client.requests",
		metric.WithDescription("Total outbound Binance API requests"),
	)
	return &otelTransport{next: next, tracer: tracer, counter: counter}
}

func (t *otelTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, span := t.tracer.Start(req.Context(), "binance.http "+req.Method+" "+req.URL.Path,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", req.Method),
			attribute.String("http.url", req.URL.String()),
		),
	)
	defer span.End()

	req = req.WithContext(ctx)
	resp, err := t.next.RoundTrip(req)

	attrs := []attribute.KeyValue{
		attribute.String("http.method", req.Method),
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		attrs = append(attrs, attribute.String("http.status", "error"))
	} else {
		attrs = append(attrs, attribute.Int("http.status_code", resp.StatusCode))
		if resp.StatusCode >= 400 {
			span.SetStatus(codes.Error, http.StatusText(resp.StatusCode))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}

	t.counter.Add(ctx, 1, metric.WithAttributes(attrs...))
	return resp, err
}
