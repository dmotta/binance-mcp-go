package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	binance "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/adshao/go-binance/v2/options"
	"github.com/mark3labs/mcp-go/server"

	"binance-mcp-go/internal/adapter"
	"binance-mcp-go/internal/config"
	"binance-mcp-go/internal/httpmw"
	"binance-mcp-go/internal/observability"
	"binance-mcp-go/internal/tools"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config error:", err)
		os.Exit(1)
	}

	obs, err := observability.Init(ctx, cfg.LogFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "observability init error:", err)
		os.Exit(1)
	}
	defer obs.Shutdown(ctx)

	log := obs.Logger

	// Middleware chain: OTel (outermost) → CB → Retry → RateLimit → transport
	transport := httpmw.Chain(
		http.DefaultTransport,
		httpmw.NewOtelTransport,
		httpmw.NewCircuitBreakerTransport,
		httpmw.NewRetryTransport,
		httpmw.NewRateLimitTransport,
	)
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   cfg.Timeout,
	}

	// Select API endpoints. These package-level flags must be set before
	// NewClient, since the clients resolve their BaseURL at construction time.
	binance.UseTestnet = cfg.Testnet
	futures.UseTestnet = cfg.Testnet

	spotClient := binance.NewClient(cfg.APIKey, cfg.SecretKey)
	spotClient.HTTPClient = httpClient

	// The futures testnet issues its own API keys, separate from spot.
	futuresClient := futures.NewClient(cfg.FuturesAPIKey, cfg.FuturesSecretKey)
	futuresClient.HTTPClient = httpClient

	// Binance has no public options testnet; in testnet mode the adapter
	// rejects options calls and this client is never exercised.
	optsClient := options.NewClient(cfg.APIKey, cfg.SecretKey)
	optsClient.HTTPClient = httpClient

	b := adapter.New(spotClient, futuresClient, optsClient, cfg.Environment())

	log.Info("binance endpoints selected", "environment", cfg.Environment(),
		"spot", spotClient.BaseURL, "futures", futuresClient.BaseURL, "options", optsClient.BaseURL)

	s := server.NewMCPServer("binance-mcp", "1.0.0")
	tools.RegisterAll(s, b)

	log.Info("binance-mcp server starting", "transport", "stdio")

	if err := server.ServeStdio(s); err != nil {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
}
