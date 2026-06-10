package tools

import (
	"context"
	"fmt"

	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCreateSpotOrder(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("create_spot_order", "Create spot market or limit orders", `{
		"type":"object",
		"required":["symbol","side","type","quantity"],
		"properties":{
			"symbol":   {"type":"string"},
			"side":     {"type":"string","enum":["BUY","SELL"]},
			"type":     {"type":"string","enum":["MARKET","LIMIT","LIMIT_MAKER"]},
			"quantity": {"type":"number"},
			"price":    {"type":"number"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		qty := getFloat(req, "quantity")
		p := port.CreateSpotOrderParams{
			Symbol:   getString(req, "symbol"),
			Side:     getString(req, "side"),
			Type:     getString(req, "type"),
			Quantity: fmt.Sprintf("%g", qty),
		}
		if price := getFloat(req, "price"); price > 0 {
			p.Price = fmt.Sprintf("%g", price)
		}
		res, err := b.CreateSpotOrder(ctx, p)
		if err != nil {
			return resultErr("create_spot_order failed: %v", err)
		}
		return resultJSON(res)
	})
}
