package tools

import (
	"context"

	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCancelOrder(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("cancel_order", "Cancel a single order", `{
		"type":"object",
		"required":["symbol","orderId"],
		"properties":{
			"symbol":  {"type":"string"},
			"orderId": {"type":"integer"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.CancelOrder(ctx, port.CancelOrderParams{
			Symbol:  getString(req, "symbol"),
			OrderID: getInt64(req, "orderId"),
		})
		if err != nil {
			return resultErr("cancel_order failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetOpenOrders(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_open_orders", "Get all open orders", `{
		"type":"object",
		"properties":{
			"symbol": {"type":"string"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetOpenOrders(ctx, port.GetOpenOrdersParams{Symbol: getString(req, "symbol")})
		if err != nil {
			return resultErr("get_open_orders failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetOrderStatus(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_order_status", "Get order status", `{
		"type":"object",
		"required":["symbol","orderId"],
		"properties":{
			"symbol":  {"type":"string"},
			"orderId": {"type":"integer"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetOrderStatus(ctx, port.GetOrderStatusParams{
			Symbol:  getString(req, "symbol"),
			OrderID: getInt64(req, "orderId"),
		})
		if err != nil {
			return resultErr("get_order_status failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerGetMyTrades(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_my_trades", "Get account trade history", `{
		"type":"object",
		"required":["symbol"],
		"properties":{
			"symbol": {"type":"string"},
			"limit":  {"type":"integer","default":100}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetMyTrades(ctx, port.GetMyTradesParams{
			Symbol: getString(req, "symbol"),
			Limit:  getInt(req, "limit", 100),
		})
		if err != nil {
			return resultErr("get_my_trades failed: %v", err)
		}
		return resultJSON(res)
	})
}

func registerCancelAllOrders(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("cancel_all_orders", "Cancel all open orders for a symbol", `{
		"type":"object",
		"required":["symbol"],
		"properties":{
			"symbol": {"type":"string"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.CancelAllOrders(ctx, port.CancelAllOrdersParams{Symbol: getString(req, "symbol")})
		if err != nil {
			return resultErr("cancel_all_orders failed: %v", err)
		}
		return resultJSON(res)
	})
}
