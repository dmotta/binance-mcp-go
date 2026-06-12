package tools

import (
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

func resultJSON(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(b)), nil
}

func resultErr(format string, args ...any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(fmt.Sprintf(format, args...)), nil
}

func getString(req mcp.CallToolRequest, key string) string {
	return req.GetString(key, "")
}

func getInt(req mcp.CallToolRequest, key string, def int) int {
	args := req.GetArguments()
	v, ok := args[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return def
}

func getFloat(req mcp.CallToolRequest, key string) float64 {
	args := req.GetArguments()
	v, ok := args[key]
	if !ok {
		return 0
	}
	if f, ok := v.(float64); ok {
		return f
	}
	return 0
}

func getBool(req mcp.CallToolRequest, key string) bool {
	args := req.GetArguments()
	v, ok := args[key].(bool)
	return ok && v
}

func getInt64(req mcp.CallToolRequest, key string) int64 {
	args := req.GetArguments()
	v, ok := args[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	}
	return 0
}
