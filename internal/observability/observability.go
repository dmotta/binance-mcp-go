package observability

import (
	"context"
	"log/slog"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

type ShutdownFunc func(ctx context.Context) error

type Providers struct {
	Logger        *slog.Logger
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
	Shutdown       ShutdownFunc
	logFile        *os.File
}

func Init(ctx context.Context, logFilePath string) (*Providers, error) {
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	logger := slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo}))

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("binance-mcp"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		logFile.Close()
		return nil, err
	}

	traceExporter, err := stdouttrace.New(stdouttrace.WithWriter(logFile), stdouttrace.WithPrettyPrint())
	if err != nil {
		logFile.Close()
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	metricExporter, err := stdoutmetric.New(stdoutmetric.WithWriter(logFile))
	if err != nil {
		_ = tp.Shutdown(ctx)
		logFile.Close()
		return nil, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	shutdown := func(ctx context.Context) error {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return logFile.Close()
	}

	return &Providers{
		Logger:        logger,
		TracerProvider: tp,
		MeterProvider:  mp,
		Shutdown:       shutdown,
		logFile:        logFile,
	}, nil
}
