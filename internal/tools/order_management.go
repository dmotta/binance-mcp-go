package tools

import (
	"context"

	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCancelOrder(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("cancel_order", "Cancel a single order (spot or futures)", `{
		"type":"object",
		"required":["symbol","orderId"],
		"properties":{
			"symbol":  {"type":"string"},
			"orderId": {"type":"integer"},
			"market":  {"type":"string","enum":["spot","futures"],"default":"spot"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		market := getString(req, "market")
		if market == "" {
			market = "spot"
		}
		res, err := b.CancelOrder(ctx, port.CancelOrderParams{
			Symbol:  getString(req, "symbol"),
			OrderID: getInt64(req, "orderId"),
			Market:  market,
		})
		if err != nil {
			return resultErr("cancel_order failed (market=%s; if the order was created on the other market, retry with the matching 'market' parameter): %v", market, err)
		}
		return resultJSON(res)
	})
}

func registerGetOpenOrders(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_open_orders", "Get all open orders (spot or futures)", `{
		"type":"object",
		"properties":{
			"symbol": {"type":"string"},
			"market": {"type":"string","enum":["spot","futures"],"default":"spot"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		market := getString(req, "market")
		if market == "" {
			market = "spot"
		}
		res, err := b.GetOpenOrders(ctx, port.GetOpenOrdersParams{
			Symbol: getString(req, "symbol"),
			Market: market,
		})
		if err != nil {
			return resultErr("get_open_orders failed (market=%s): %v", market, err)
		}
		return resultJSON(res)
	})
}

func registerGetOrderStatus(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_order_status", "Get order status (spot or futures)", `{
		"type":"object",
		"required":["symbol","orderId"],
		"properties":{
			"symbol":  {"type":"string"},
			"orderId": {"type":"integer"},
			"market":  {"type":"string","enum":["spot","futures"],"default":"spot"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		market := getString(req, "market")
		if market == "" {
			market = "spot"
		}
		res, err := b.GetOrderStatus(ctx, port.GetOrderStatusParams{
			Symbol:  getString(req, "symbol"),
			OrderID: getInt64(req, "orderId"),
			Market:  market,
		})
		if err != nil {
			return resultErr("get_order_status failed (market=%s; if the order was created on the other market, retry with the matching 'market' parameter): %v", market, err)
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
	t := toolWithRawSchema("cancel_all_orders", "Cancel all open orders for a symbol (spot or futures)", `{
		"type":"object",
		"required":["symbol"],
		"properties":{
			"symbol": {"type":"string"},
			"market": {"type":"string","enum":["spot","futures"],"default":"spot"}
		}
	}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		market := getString(req, "market")
		if market == "" {
			market = "spot"
		}
		res, err := b.CancelAllOrders(ctx, port.CancelAllOrdersParams{
			Symbol: getString(req, "symbol"),
			Market: market,
		})
		if err != nil {
			return resultErr("cancel_all_orders failed (market=%s): %v", market, err)
		}
		return resultJSON(res)
	})
}
