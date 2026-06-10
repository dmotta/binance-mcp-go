package tools

import (
	"context"
	"fmt"

	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCreateOptionOrder(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("create_option_order", "Create an options order", `{
		"type":"object",
		"required":["symbol","side","quantity","price"],
		"properties":{
			"symbol":   {"type":"string"},
			"side":     {"type":"string","enum":["BUY","SELL"]},
			"quantity": {"type":"number"},
			"price":    {"type":"number"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.CreateOptionOrder(ctx, port.OptionOrderParams{
			Symbol:   getString(req, "symbol"),
			Side:     getString(req, "side"),
			Quantity: fmt.Sprintf("%g", getFloat(req, "quantity")),
			Price:    fmt.Sprintf("%g", getFloat(req, "price")),
		})
		if err != nil {
			return resultErr("create_option_order failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetOptionChain(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_option_chain", "Get option chain for an underlying asset", `{
		"type":"object",
		"required":["underlying"],
		"properties":{
			"underlying": {"type":"string"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetOptionChain(ctx, getString(req, "underlying"))
		if err != nil {
			return resultErr("get_option_chain failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetOptionPositions(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_option_positions", "Get all open option positions", `{"type":"object","properties":{}}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetOptionPositions(ctx)
		if err != nil {
			return resultErr("get_option_positions failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetOptionInfo(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_option_info", "Get details for a specific option symbol", `{
		"type":"object",
		"required":["symbol"],
		"properties":{
			"symbol": {"type":"string"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetOptionInfo(ctx, getString(req, "symbol"))
		if err != nil {
			return resultErr("get_option_info failed: %v", err)
		}
		return resultJSON(res)
	})
}
