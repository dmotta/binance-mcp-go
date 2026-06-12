package tools

import (
	"context"
	"fmt"

	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetBalance(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_balance", "Get spot account balances", `{
		"type":"object",
		"properties":{
			"asset": {"type":"string"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetBalance(ctx, getString(req, "asset"))
		if err != nil {
			return resultErr("get_balance failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetFuturesBalance(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_futures_balance", "Get USDⓈ-M futures wallet balances", `{
		"type":"object",
		"properties":{
			"asset": {"type":"string","description":"Optional; restrict the result to one asset, e.g. USDT"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetFuturesBalance(ctx, getString(req, "asset"))
		if err != nil {
			return resultErr("get_futures_balance failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetPositions(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_positions", "Get open futures positions for the account", `{"type":"object","properties":{}}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetPositions(ctx)
		if err != nil {
			return resultErr("get_positions failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerSetLeverage(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("set_leverage", "Set leverage for a futures symbol", `{
		"type":"object",
		"required":["symbol","leverage"],
		"properties":{
			"symbol":   {"type":"string"},
			"leverage": {"type":"integer","minimum":1,"maximum":125}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		err := b.SetLeverage(ctx, port.SetLeverageParams{
			Symbol:   getString(req, "symbol"),
			Leverage: getInt(req, "leverage", 1),
		})
		if err != nil {
			return resultErr("set_leverage failed: %v", err)
		}
		return resultJSON(map[string]string{"status": "ok"})
	})
}

func registerSetMarginMode(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("set_margin_mode", "Set margin mode (ISOLATED or CROSSED) for a futures symbol", `{
		"type":"object",
		"required":["symbol","marginMode"],
		"properties":{
			"symbol":     {"type":"string"},
			"marginMode": {"type":"string","enum":["ISOLATED","CROSSED"]}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		err := b.SetMarginMode(ctx, port.SetMarginModeParams{
			Symbol:     getString(req, "symbol"),
			MarginMode: getString(req, "marginMode"),
		})
		if err != nil {
			return resultErr("set_margin_mode failed: %v", err)
		}
		return resultJSON(map[string]string{"status": "ok"})
	})
}

func registerTransferFunds(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("transfer_funds", "Transfer funds between account types", `{
		"type":"object",
		"required":["asset","amount","fromAccount","toAccount"],
		"properties":{
			"asset":       {"type":"string"},
			"amount":      {"type":"number"},
			"fromAccount": {"type":"string","enum":["SPOT","FUNDING","MARGIN","FUTURES"]},
			"toAccount":   {"type":"string","enum":["SPOT","FUNDING","MARGIN","FUTURES"]}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		err := b.TransferFunds(ctx, port.TransferFundsParams{
			Asset:       getString(req, "asset"),
			Amount:      fmt.Sprintf("%g", getFloat(req, "amount")),
			FromAccount: getString(req, "fromAccount"),
			ToAccount:   getString(req, "toAccount"),
		})
		if err != nil {
			return resultErr("transfer_funds failed: %v", err)
		}
		return resultJSON(map[string]string{"status": "ok"})
	})
}
