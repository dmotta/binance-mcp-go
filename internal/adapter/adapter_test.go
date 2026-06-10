package adapter

import (
	"context"
	"strings"
	"testing"

	binance "github.com/adshao/go-binance/v2"
	"github.com/adshao/go-binance/v2/options"

	"binance-mcp-go/internal/port"
)

func TestResolveTransferType(t *testing.T) {
	cases := []struct {
		from, to string
		want     binance.UserUniversalTransferType
	}{
		{"SPOT", "FUTURES", binance.UserUniversalTransferTypeMainToUmFutures},
		{"SPOT", "MARGIN", binance.UserUniversalTransferTypeMainToMargin},
		{"SPOT", "FUNDING", binance.UserUniversalTransferTypeMainToFunding},
		{"FUTURES", "SPOT", binance.UserUniversalTransferTypeUmFuturesToMain},
		{"FUTURES", "MARGIN", binance.UserUniversalTransferTypeUmFuturesToMargin},
		{"FUTURES", "FUNDING", binance.UserUniversalTransferTypeUmFuturesToFunding},
		{"MARGIN", "SPOT", binance.UserUniversalTransferTypeMarginToMain},
		{"MARGIN", "FUTURES", binance.UserUniversalTransferTypeMarginToUmFutures},
		{"MARGIN", "FUNDING", binance.UserUniversalTransferTypeMarginToFunding},
		{"FUNDING", "SPOT", binance.UserUniversalTransferTypeFundingToMain},
		{"FUNDING", "FUTURES", binance.UserUniversalTransferTypeFundingToUmFutures},
		// Regression: this pair was inverted to MarginToFunding before the fix.
		{"FUNDING", "MARGIN", binance.UserUniversalTransferTypeFundingToMargin},
	}
	for _, c := range cases {
		got, err := resolveTransferType(c.from, c.to)
		if err != nil {
			t.Fatalf("%s→%s: unexpected error %v", c.from, c.to, err)
		}
		if got != c.want {
			t.Fatalf("%s→%s: got %q want %q", c.from, c.to, got, c.want)
		}
	}
}

func TestResolveTransferType_Unsupported(t *testing.T) {
	if _, err := resolveTransferType("SPOT", "BANK"); err == nil {
		t.Fatal("expected error for unsupported transfer pair")
	}
}

func TestCreateOptionOrder_RequiresPrice(t *testing.T) {
	a := New(nil, nil, options.NewClient("", ""), "production")
	// Missing price must be rejected before any network call (clients are unconfigured).
	_, err := a.CreateOptionOrder(context.Background(), port.OptionOrderParams{
		Symbol:   "BTC-240927-60000-C",
		Side:     "BUY",
		Quantity: "1",
		Price:    "",
	})
	if err == nil {
		t.Fatal("expected error when price is empty")
	}
	if !strings.Contains(err.Error(), "price") {
		t.Fatalf("expected price error, got: %v", err)
	}
}

func TestOptions_UnavailableOnTestnet(t *testing.T) {
	// Binance has no public options testnet; every options entry point must
	// fail fast with a clear message instead of dialing a nonexistent host.
	a := New(nil, nil, options.NewClient("", ""), "testnet")
	ctx := context.Background()

	calls := map[string]func() error{
		"CreateOptionOrder": func() error {
			_, err := a.CreateOptionOrder(ctx, port.OptionOrderParams{
				Symbol: "BTC-240927-60000-C", Side: "BUY", Quantity: "1", Price: "100",
			})
			return err
		},
		"GetOptionChain":     func() error { _, err := a.GetOptionChain(ctx, "BTC"); return err },
		"GetOptionPositions": func() error { _, err := a.GetOptionPositions(ctx); return err },
		"GetOptionInfo":      func() error { _, err := a.GetOptionInfo(ctx, "BTC-240927-60000-C"); return err },
	}
	for name, call := range calls {
		err := call()
		if err == nil {
			t.Fatalf("%s: expected testnet error, got nil", name)
		}
		if !strings.Contains(err.Error(), "testnet") {
			t.Fatalf("%s: expected testnet error, got: %v", name, err)
		}
	}
}
