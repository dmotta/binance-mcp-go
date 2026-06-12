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
		"required":["symbol","side","type"],
		"properties":{
			"symbol":        {"type":"string"},
			"side":          {"type":"string","enum":["BUY","SELL"]},
			"type":          {"type":"string","enum":["MARKET","LIMIT","STOP_MARKET","TAKE_PROFIT_MARKET"]},
			"quantity":      {"type":"number","description":"Required unless closePosition is true"},
			"price":         {"type":"number","description":"Required for LIMIT orders"},
			"stopPrice":     {"type":"number","description":"Trigger price; required for STOP_MARKET and TAKE_PROFIT_MARKET"},
			"reduceOnly":    {"type":"boolean","description":"Only reduce an existing position, never open or flip one"},
			"closePosition": {"type":"boolean","description":"Close the entire position when triggered (STOP_MARKET/TAKE_PROFIT_MARKET only); omit quantity and reduceOnly"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		p := port.ContractOrderParams{
			Symbol:        getString(req, "symbol"),
			Side:          getString(req, "side"),
			Type:          getString(req, "type"),
			ReduceOnly:    getBool(req, "reduceOnly"),
			ClosePosition: getBool(req, "closePosition"),
		}
		if qty := getFloat(req, "quantity"); qty > 0 {
			p.Quantity = fmt.Sprintf("%g", qty)
		}
		if price := getFloat(req, "price"); price > 0 {
			p.Price = fmt.Sprintf("%g", price)
		}
		if stopPrice := getFloat(req, "stopPrice"); stopPrice > 0 {
			p.StopPrice = fmt.Sprintf("%g", stopPrice)
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
	t := toolWithRawSchema("get_futures_positions", "Get open futures positions, optionally filtered by symbol", `{
		"type":"object",
		"properties":{
			"symbol": {"type":"string","description":"Optional; restrict the result to one symbol"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetFuturesPositions(ctx, getString(req, "symbol"))
		if err != nil {
			return resultErr("get_futures_positions failed: %v", err)
		}
		return resultJSON(res)
	})
}
