package server

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/freeDog-wy/go-backend-template/internal/app/mcp/contract"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func toolOutput(data json.RawMessage, err error) (*mcp.CallToolResult, map[string]any, error) {
	if err != nil {
		return toolFailure(err), nil, nil
	}
	output, err := rawObject(data)
	if err != nil {
		return toolFailure(err), nil, nil
	}
	return nil, output, nil
}

func rawObject(data json.RawMessage) (map[string]any, error) {
	var output map[string]any
	if err := json.Unmarshal(data, &output); err != nil {
		return nil, fmt.Errorf("decode CMS response: %w", err)
	}
	return output, nil
}

func toolFailure(err error) *mcp.CallToolResult {
	var apiErr *contract.APIError
	if errors.As(err, &apiErr) {
		return toolError(apiErr.Code, apiErr.Message)
	}
	return toolError("CMS_UNAVAILABLE", "CMS request failed")
}

func toolError(code, message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: code + ": " + message}}}
}
