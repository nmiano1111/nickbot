// Package model defines a provider-agnostic interface for chat models.
//
// Nothing outside this package should depend on a specific backend's
// request/response shapes (Ollama, Anthropic, OpenAI, ...). Backends live in
// this package as separate files (ollama.go, ...) and adapt their wire
// format to these types.
package model

import (
	"context"
	"encoding/json"
)

// ToolCall is a request from the model to invoke one tool.
type ToolCall struct {
	// ID correlates a call with its result for backends that require it
	// (e.g. OpenAI). Ollama does not, and may leave it empty.
	ID        string
	Name      string
	Arguments json.RawMessage
}

// Message is one turn in a chat conversation.
type Message struct {
	Role    string
	Content string
	// Thinking holds a reasoning model's intermediate "thinking" tokens,
	// separate from its final Content. Not all backends or models set it.
	Thinking string
	// ToolCalls is set on an assistant message that is requesting tool
	// invocations instead of (or in addition to) a text answer.
	ToolCalls []ToolCall
}

// ToolDefinition describes one callable tool to the model, independent of
// any tools.Tool implementation — this package does not depend on the
// tools package.
type ToolDefinition struct {
	Name        string
	Description string
	// Parameters is a JSON Schema object describing the tool's arguments.
	Parameters map[string]any
}

// ChatRequest is a single chat exchange: the conversation so far, the
// tools available this turn, plus optional callbacks a backend invokes
// incrementally while streaming a reply. Either callback may be nil.
type ChatRequest struct {
	Messages   []Message
	Tools      []ToolDefinition
	OnThinking func(token string)
	OnContent  func(token string)
}

// ChatResponse is the result of a chat exchange.
type ChatResponse struct {
	Message Message
}

// Model is anything that can carry out a chat exchange. The agent runtime
// should depend only on this interface, never on a specific backend.
type Model interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}
