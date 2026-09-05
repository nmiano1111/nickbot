// Package tools defines the agent's tool abstraction: a small interface
// each capability (read_file, grep, ...) implements, plus a registry the
// agent loop dispatches through generically instead of switching on name.
package tools

import (
	"context"
	"encoding/json"
)

// ToolResult is what a Tool returns after execution.
type ToolResult struct {
	Content string
	IsError bool
}

// Tool is a single capability the agent can invoke. Schema returns a JSON
// Schema object describing Execute's expected args, suitable for handing to
// a model's structured tool-calling API.
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Execute(ctx context.Context, args json.RawMessage) (ToolResult, error)
}
