package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Ollama is a Model backed by a locally running Ollama server.
type Ollama struct {
	Host       string
	ModelName  string
	HTTPClient *http.Client
}

// NewOllama returns an Ollama model client. host is the server base URL
// (e.g. "http://localhost:11434") and modelName is the model to chat with
// (e.g. "qwen3.5:9b").
func NewOllama(host, modelName string) *Ollama {
	return &Ollama{
		Host:       host,
		ModelName:  modelName,
		HTTPClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

// wire types for Ollama's /api/chat, kept private so no Ollama-specific
// shape leaks outside this file.

type ollamaMessage struct {
	Role     string `json:"role"`
	Content  string `json:"content"`
	Thinking string `json:"thinking,omitempty"`
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
	Error   string        `json:"error"`
}

// Chat implements Model.
func (o *Ollama) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	messages := make([]ollamaMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = ollamaMessage{Role: msg.Role, Content: msg.Content}
	}

	body, err := json.Marshal(ollamaChatRequest{Model: o.ModelName, Messages: messages, Stream: true})
	if err != nil {
		return ChatResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Host+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.HTTPClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("request to ollama failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return ChatResponse{}, fmt.Errorf("ollama returned %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var full strings.Builder
	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk ollamaChatResponse
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return ChatResponse{Message: Message{Role: "assistant", Content: full.String()}},
				fmt.Errorf("reading stream: %w", err)
		}
		if chunk.Error != "" {
			return ChatResponse{Message: Message{Role: "assistant", Content: full.String()}},
				fmt.Errorf("ollama error: %s", chunk.Error)
		}
		if chunk.Message.Thinking != "" && req.OnThinking != nil {
			req.OnThinking(chunk.Message.Thinking)
		}
		if chunk.Message.Content != "" {
			full.WriteString(chunk.Message.Content)
			if req.OnContent != nil {
				req.OnContent(chunk.Message.Content)
			}
		}
		if chunk.Done {
			break
		}
	}

	return ChatResponse{Message: Message{Role: "assistant", Content: full.String()}}, nil
}
