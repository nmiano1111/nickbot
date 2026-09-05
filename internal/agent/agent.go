// Package agent implements the tool-use loop: it drives model.Model and
// tools.Registry together so the model can autonomously decide to call
// tools, see their results, and keep going until it has a final answer.
//
// This package is the only place that knows about both model and tools —
// neither of those packages knows about the other.
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"nickbot/internal/model"
	"nickbot/internal/tools"
)

// defaultMaxTurns bounds how many model/tool round trips a single Run call
// will make before giving up, so a confused model can't loop forever.
const defaultMaxTurns = 8

// Agent orchestrates a model and a tool registry across a multi-turn tool
// loop. All callback fields are optional and, if set, are invoked
// synchronously as the loop progresses (useful for streaming UI feedback).
type Agent struct {
	Model    model.Model
	Tools    *tools.Registry
	MaxTurns int

	// OnTurnStart fires before each model.Chat call in the loop.
	OnTurnStart func()
	// OnThinking and OnContent fire incrementally as the model streams a
	// reply for the current turn.
	OnThinking func(token string)
	OnContent  func(token string)
	// OnToolCall fires once per tool call the model requests.
	OnToolCall func(name string, args json.RawMessage)
	// OnToolResult fires once a tool call has been executed.
	OnToolResult func(name string, result tools.ToolResult)
}

// Run drives the tool loop for one user turn. It appends every
// assistant/tool message produced along the way to *history in place, and
// returns the final answer text.
func (a *Agent) Run(ctx context.Context, history *[]model.Message) (string, error) {
	maxTurns := a.MaxTurns
	if maxTurns <= 0 {
		maxTurns = defaultMaxTurns
	}

	for range maxTurns {
		if a.OnTurnStart != nil {
			a.OnTurnStart()
		}

		resp, err := a.Model.Chat(ctx, model.ChatRequest{
			Messages:   *history,
			Tools:      a.toolDefinitions(),
			OnThinking: a.OnThinking,
			OnContent:  a.OnContent,
		})
		if err != nil {
			return "", err
		}

		*history = append(*history, resp.Message)

		if len(resp.Message.ToolCalls) == 0 {
			return resp.Message.Content, nil
		}

		for _, call := range resp.Message.ToolCalls {
			if a.OnToolCall != nil {
				a.OnToolCall(call.Name, call.Arguments)
			}
			result := a.executeTool(ctx, call)
			if a.OnToolResult != nil {
				a.OnToolResult(call.Name, result)
			}
			*history = append(*history, model.Message{Role: "tool", Content: result.Content})
		}
	}

	return "", fmt.Errorf("maximum agent turns (%d) exceeded", maxTurns)
}

func (a *Agent) toolDefinitions() []model.ToolDefinition {
	list := a.Tools.List()
	if len(list) == 0 {
		return nil
	}
	defs := make([]model.ToolDefinition, len(list))
	for i, t := range list {
		defs[i] = model.ToolDefinition{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Schema(),
		}
	}
	return defs
}

func (a *Agent) executeTool(ctx context.Context, call model.ToolCall) tools.ToolResult {
	tool, ok := a.Tools.Get(call.Name)
	if !ok {
		return tools.ToolResult{Content: fmt.Sprintf("unknown tool %q", call.Name), IsError: true}
	}
	result, err := tool.Execute(ctx, call.Arguments)
	if err != nil {
		return tools.ToolResult{Content: fmt.Sprintf("tool %q failed: %v", call.Name, err), IsError: true}
	}
	return result
}
