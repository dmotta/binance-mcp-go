// Package tools registers all MCP tools against a server instance.
package tools

import (
	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/server"
)

// RegisterAll registers every tool defined in mcp-tools.yaml onto s.
func RegisterAll(s *server.MCPServer, b port.BinancePort) {
	registerCreateSpotOrder(s, b)
	registerCancelOrder(s, b)
	registerGetOpenOrders(s, b)
	registerGetOrderStatus(s, b)
	registerGetMyTrades(s, b)
	registerCancelAllOrders(s, b)
	registerCreateStopLossOrder(s, b)
	registerCreateTakeProfitOrder(s, b)
	registerCreateStopLimitOrder(s, b)
	registerCreateTrailingStopOrder(s, b)
	registerCreateOCOOrder(s, b)
	registerGetTicker(s, b)
	registerGetOrderBook(s, b)
	registerGetKlines(s, b)
	registerGetFundingRate(s, b)
	registerCreateOptionOrder(s, b)
	registerGetOptionChain(s, b)
	registerGetOptionPositions(s, b)
	registerGetOptionInfo(s, b)
	registerCreateContractOrder(s, b)
	registerClosePosition(s, b)
	registerGetFuturesPositions(s, b)
	registerGetBalance(s, b)
	registerGetFuturesBalance(s, b)
	registerGetPositions(s, b)
	registerSetLeverage(s, b)
	registerSetMarginMode(s, b)
	registerTransferFunds(s, b)
	registerGetServerInfo(s, b)
}
