package redact

import (
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestURL_RedactsSignatureAndKeepsRest(t *testing.T) {
	raw := "https://api.binance.com/api/v3/order?symbol=BTCUSDT&side=BUY&timestamp=123&signature=deadbeefcafe"
	got := URL(raw)

	if strings.Contains(got, "deadbeefcafe") {
		t.Fatalf("signature value leaked: %q", got)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("redacted URL no longer parses: %v", err)
	}
	q := u.Query()
	if q.Get("signature") != placeholder {
		t.Fatalf("signature not redacted, got %q", q.Get("signature"))
	}
	if q.Get("symbol") != "BTCUSDT" || q.Get("side") != "BUY" {
		t.Fatalf("non-sensitive params were altered: %q", got)
	}
}

func TestURL_RedactsApiKey(t *testing.T) {
	got := URL("https://x/y?apiKey=SECRET123&a=b")
	if strings.Contains(got, "SECRET123") {
		t.Fatalf("apiKey leaked: %q", got)
	}
}

func TestURL_NoSensitiveParamsUnchanged(t *testing.T) {
	raw := "https://api.binance.com/api/v3/klines?symbol=BTCUSDT&interval=4h"
	if got := URL(raw); got != raw {
		t.Fatalf("URL without secrets should be unchanged, got %q", got)
	}
}

func TestString_RedactsEmbeddedSignature(t *testing.T) {
	// Shape of an *url.Error surfaced by http.Client on a transport failure.
	msg := `Post "https://api.binance.com/api/v3/order?symbol=BTCUSDT&signature=abc123def": dial tcp: timeout`
	got := String(msg)
	if strings.Contains(got, "abc123def") {
		t.Fatalf("signature leaked in error string: %q", got)
	}
	if !strings.Contains(got, "signature="+placeholder) {
		t.Fatalf("expected redacted marker, got %q", got)
	}
	if !strings.Contains(got, "symbol=BTCUSDT") {
		t.Fatalf("redaction removed non-sensitive context: %q", got)
	}
}

func TestError_NilSafe(t *testing.T) {
	if Error(nil) != "" {
		t.Fatal("Error(nil) must be empty")
	}
	if got := Error(errors.New("signature=xyz failed")); strings.Contains(got, "xyz") {
		t.Fatalf("signature leaked: %q", got)
	}
}
