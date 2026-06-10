package tools

import (
	"context"
	"fmt"

	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCreateStopLossOrder(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("create_stop_loss_order", "Create stop loss order", `{
		"type":"object",
		"required":["symbol","side","quantity","stopPrice"],
		"properties":{
			"symbol":    {"type":"string"},
			"side":      {"type":"string","enum":["BUY","SELL"]},
			"quantity":  {"type":"number"},
			"stopPrice": {"type":"number"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.CreateStopLossOrder(ctx, port.StopOrderParams{
			Symbol:    getString(req, "symbol"),
			Side:      getString(req, "side"),
			Quantity:  fmt.Sprintf("%g", getFloat(req, "quantity")),
			StopPrice: fmt.Sprintf("%g", getFloat(req, "stopPrice")),
		})
		if err != nil {
			return resultErr("create_stop_loss_order failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerCreateTakeProfitOrder(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("create_take_profit_order", "Create take profit order", `{
		"type":"object",
		"required":["symbol","side","quantity","stopPrice"],
		"properties":{
			"symbol":    {"type":"string"},
			"side":      {"type":"string","enum":["BUY","SELL"]},
			"quantity":  {"type":"number"},
			"stopPrice": {"type":"number"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.CreateTakeProfitOrder(ctx, port.StopOrderParams{
			Symbol:    getString(req, "symbol"),
			Side:      getString(req, "side"),
			Quantity:  fmt.Sprintf("%g", getFloat(req, "quantity")),
			StopPrice: fmt.Sprintf("%g", getFloat(req, "stopPrice")),
		})
		if err != nil {
			return resultErr("create_take_profit_order failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerCreateStopLimitOrder(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("create_stop_limit_order", "Create stop limit order", `{
		"type":"object",
		"required":["symbol","side","quantity","stopPrice","limitPrice"],
		"properties":{
			"symbol":     {"type":"string"},
			"side":       {"type":"string","enum":["BUY","SELL"]},
			"quantity":   {"type":"number"},
			"stopPrice":  {"type":"number"},
			"limitPrice": {"type":"number"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.CreateStopLimitOrder(ctx, port.StopLimitOrderParams{
			Symbol:     getString(req, "symbol"),
			Side:       getString(req, "side"),
			Quantity:   fmt.Sprintf("%g", getFloat(req, "quantity")),
			StopPrice:  fmt.Sprintf("%g", getFloat(req, "stopPrice")),
			LimitPrice: fmt.Sprintf("%g", getFloat(req, "limitPrice")),
		})
		if err != nil {
			return resultErr("create_stop_limit_order failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerCreateTrailingStopOrder(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("create_trailing_stop_order", "Create trailing stop order. callbackRate is a PERCENTAGE (1.5 = 1.5%), valid range 0.1-20; the server converts it to BIPS (trailingDelta) internally — do NOT pre-convert.", `{
		"type":"object",
		"required":["symbol","side","quantity","callbackRate"],
		"properties":{
			"symbol":       {"type":"string"},
			"side":         {"type":"string","enum":["BUY","SELL"]},
			"quantity":     {"type":"number"},
			"callbackRate": {"type":"number","minimum":0.1,"maximum":20,"description":"Trailing distance as a percent of price (unit: percent, NOT BIPS). Example: 1.5 means 1.5% (sent to Binance as trailingDelta=150 BIPS). Valid range: 0.1 to 20."}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.CreateTrailingStopOrder(ctx, port.TrailingStopOrderParams{
			Symbol:       getString(req, "symbol"),
			Side:         getString(req, "side"),
			Quantity:     fmt.Sprintf("%g", getFloat(req, "quantity")),
			CallbackRate: getFloat(req, "callbackRate"),
		})
		if err != nil {
			return resultErr("create_trailing_stop_order failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerCreateOCOOrder(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("create_oco_order", "Create OCO order", `{
		"type":"object",
		"required":["symbol","side","quantity","price","stopPrice"],
		"properties":{
			"symbol":         {"type":"string"},
			"side":           {"type":"string","enum":["BUY","SELL"]},
			"quantity":       {"type":"number"},
			"price":          {"type":"number"},
			"stopPrice":      {"type":"number"},
			"stopLimitPrice": {"type":"number"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		p := port.OCOOrderParams{
			Symbol:    getString(req, "symbol"),
			Side:      getString(req, "side"),
			Quantity:  fmt.Sprintf("%g", getFloat(req, "quantity")),
			Price:     fmt.Sprintf("%g", getFloat(req, "price")),
			StopPrice: fmt.Sprintf("%g", getFloat(req, "stopPrice")),
		}
		if slp := getFloat(req, "stopLimitPrice"); slp > 0 {
			p.StopLimitPrice = fmt.Sprintf("%g", slp)
		}
		res, err := b.CreateOCOOrder(ctx, p)
		if err != nil {
			return resultErr("create_oco_order failed: %v", err)
		}
		return resultJSON(res)
	})
}
