package tools

import (
	"context"

	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetTicker(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_ticker", "Get latest ticker", `{
		"type":"object",
		"required":["symbol"],
		"properties":{
			"symbol": {"type":"string"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetTicker(ctx, getString(req, "symbol"))
		if err != nil {
			return resultErr("get_ticker failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetOrderBook(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_order_book", "Get market depth", `{
		"type":"object",
		"required":["symbol"],
		"properties":{
			"symbol": {"type":"string"},
			"limit":  {"type":"integer","default":100}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetOrderBook(ctx, getString(req, "symbol"), getInt(req, "limit", 100))
		if err != nil {
			return resultErr("get_order_book failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetKlines(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_klines", "Get candlestick data", `{
		"type":"object",
		"required":["symbol","interval"],
		"properties":{
			"symbol":   {"type":"string"},
			"interval": {"type":"string","enum":["1m","3m","5m","15m","30m","1h","4h","1d","1w","1M"]},
			"limit":    {"type":"integer","default":500}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetKlines(ctx,
			getString(req, "symbol"),
			getString(req, "interval"),
			getInt(req, "limit", 500),
		)
		if err != nil {
			return resultErr("get_klines failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetFundingRate(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_funding_rate", "Get perpetual funding rate", `{
		"type":"object",
		"properties":{
			"symbol": {"type":"string"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetFundingRate(ctx, getString(req, "symbol"))
		if err != nil {
			return resultErr("get_funding_rate failed: %v", err)
		}
		return resultJSON(res)
	})
}
