package tools

import (
	"context"

	"binance-mcp-go/internal/port"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerGetServerInfo(s *server.MCPServer, b port.BinancePort) {
	t := toolWithRawSchema("get_server_info", "Get Binance server time, version and environment", `{"type":"object","properties":{}}`)
	s.AddTool(t, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		res, err := b.GetServerInfo(ctx)
		if err != nil {
			return resultErr("get_server_info failed: %v", err)
		}
		return resultJSON(res)
	})
}
