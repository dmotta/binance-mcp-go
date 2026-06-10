package tools

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// toolWithRawSchema builds a mcp.Tool with a raw JSON input schema.
func toolWithRawSchema(name, description string, schema string) mcp.Tool {
	return mcp.NewToolWithRawSchema(name, description, json.RawMessage(schema))
}
