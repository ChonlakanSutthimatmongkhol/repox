// Package mcp exposes Repox functionality as an MCP (Model Context Protocol) server.
package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Serve starts the MCP server on stdio and blocks until the client disconnects.
func Serve() error {
	s := server.NewMCPServer(
		"repox",
		"1.0.2",
		server.WithToolCapabilities(true),
	)

	registerTools(s)

	if err := server.ServeStdio(s); err != nil {
		return fmt.Errorf("mcp: serve: %w", err)
	}
	return nil
}

// registerTools adds all Repox tool definitions to the server.
func registerTools(s *server.MCPServer) {
	s.AddTool(toolScan(), handleScan)
	s.AddTool(toolGenerate(), handleGenerate)
	s.AddTool(toolFindSimilar(), handleFindSimilar)
	s.AddTool(toolLearn(), handleLearn)
	s.AddTool(toolExplainConvention(), handleExplainConvention)
}

// callResult is a convenience helper to return a plain text result.
func callResult(text string) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(text), nil
}

func callError(err error) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(err.Error()), nil
}

// argsMap safely casts the raw arguments interface to map[string]any.
func argsMap(raw any) map[string]any {
	m, _ := raw.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

// optionalString extracts an optional string field from MCP tool arguments.
func optionalString(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func optionalBool(args map[string]any, key string) bool {
	v, _ := args[key].(bool)
	return v
}

func optionalInt(args map[string]any, key, _ string) int {
	v, ok := args[key].(float64) // JSON numbers come as float64
	if !ok {
		return 3
	}
	return int(v)
}

// unused context arg silencer
var _ = context.Background
