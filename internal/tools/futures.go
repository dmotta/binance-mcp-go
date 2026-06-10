package tools

import (
	"context"
	"fmt"

	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCreateContractOrder(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("create_contract_order", "Create a futures/contract order", `{
		"type":"object",
		"required":["symbol","side","type","quantity"],
		"properties":{
			"symbol":   {"type":"string"},
			"side":     {"type":"string","enum":["BUY","SELL"]},
			"type":     {"type":"string","enum":["MARKET","LIMIT","STOP","TAKE_PROFIT"]},
			"quantity": {"type":"number"},
			"price":    {"type":"number"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		p := port.ContractOrderParams{
			Symbol:   getString(req, "symbol"),
			Side:     getString(req, "side"),
			Type:     getString(req, "type"),
			Quantity: fmt.Sprintf("%g", getFloat(req, "quantity")),
		}
		if price := getFloat(req, "price"); price > 0 {
			p.Price = fmt.Sprintf("%g", price)
		}
		res, err := b.CreateContractOrder(ctx, p)
		if err != nil {
			return resultErr("create_contract_order failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerClosePosition(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("close_position", "Close entire futures position for a symbol", `{
		"type":"object",
		"required":["symbol"],
		"properties":{
			"symbol": {"type":"string"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.ClosePosition(ctx, getString(req, "symbol"))
		if err != nil {
			return resultErr("close_position failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetFuturesPositions(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_futures_positions", "Get all open futures positions", `{"type":"object","properties":{}}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetFuturesPositions(ctx)
		if err != nil {
			return resultErr("get_futures_positions failed: %v", err)
		}
		return resultJSON(res)
	})
}
