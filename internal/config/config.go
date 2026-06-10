package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	APIKey    string
	SecretKey string
	// The futures testnet (testnet.binancefuture.com) issues its own API keys,
	// independent from the spot testnet. These default to APIKey/SecretKey.
	FuturesAPIKey    string
	FuturesSecretKey string
	LogFile          string
	Timeout          time.Duration
	Testnet          bool
}

// Environment returns the human-readable environment name reported by the
// get_server_info tool.
func (c *Config) Environment() string {
	if c.Testnet {
		return "testnet"
	}
	return "production"
}

func Load() (*Config, error) {
	apiKey := os.Getenv("BINANCE_API_KEY")
	secretKey := os.Getenv("BINANCE_SECRET_KEY")

	if apiKey == "" {
		return nil, fmt.Errorf("BINANCE_API_KEY is required")
	}
	if secretKey == "" {
		return nil, fmt.Errorf("BINANCE_SECRET_KEY is required")
	}

	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = defaultLogPath()
	}

	timeoutSecs := 30
	if v := os.Getenv("TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeoutSecs = n
		}
	}

	testnet := false
	if v := os.Getenv("BINANCE_TESTNET"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			testnet = b
		}
	}

	futAPIKey := os.Getenv("BINANCE_FUTURES_API_KEY")
	if futAPIKey == "" {
		futAPIKey = apiKey
	}
	futSecretKey := os.Getenv("BINANCE_FUTURES_SECRET_KEY")
	if futSecretKey == "" {
		futSecretKey = secretKey
	}

	return &Config{
		APIKey:           apiKey,
		SecretKey:        secretKey,
		FuturesAPIKey:    futAPIKey,
		FuturesSecretKey: futSecretKey,
		LogFile:          logFile,
		Timeout:          time.Duration(timeoutSecs) * time.Second,
		Testnet:          testnet,
	}, nil
}

func defaultLogPath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "binance-mcp.log")
	}
	dir := filepath.Join(cacheDir, "binance-mcp")
	_ = os.MkdirAll(dir, 0o700)
	return filepath.Join(dir, "binance-mcp.log")
}
