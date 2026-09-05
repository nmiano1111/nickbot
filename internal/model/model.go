// Package model defines a provider-agnostic interface for chat models.
//
// Nothing outside this package should depend on a specific backend's
// request/response shapes (Ollama, Anthropic, OpenAI, ...). Backends live in
// this package as separate files (ollama.go, ...) and adapt their wire
// format to these types.
package model

import "context"

// Message is one turn in a chat conversation.
type Message struct {
	Role    string
	Content string
	// Thinking holds a reasoning model's intermediate "thinking" tokens,
	// separate from its final Content. Not all backends or models set it.
	Thinking string
}

// ChatRequest is a single chat exchange: the conversation so far, plus
// optional callbacks a backend invokes incrementally while streaming a
// reply. Either callback may be nil.
type ChatRequest struct {
	Messages   []Message
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
