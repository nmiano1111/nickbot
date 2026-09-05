package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"nickbot/internal/model"
	"nickbot/internal/tools"
)

// fakeModel returns one canned response per call, in order.
type fakeModel struct {
	responses []model.ChatResponse
	calls     int
}

func (f *fakeModel) Chat(_ context.Context, _ model.ChatRequest) (model.ChatResponse, error) {
	if f.calls >= len(f.responses) {
		return model.ChatResponse{}, nil
	}
	resp := f.responses[f.calls]
	f.calls++
	return resp, nil
}

// echoTool returns whatever args it was given, for asserting the loop
// dispatched correctly.
type echoTool struct{ name string }

func (e echoTool) Name() string           { return e.name }
func (e echoTool) Description() string    { return "echoes its args" }
func (e echoTool) Schema() map[string]any { return map[string]any{"type": "object"} }
func (e echoTool) Execute(_ context.Context, args json.RawMessage) (tools.ToolResult, error) {
	return tools.ToolResult{Content: "echo:" + string(args)}, nil
}

func TestRunAnswersWithoutToolCalls(t *testing.T) {
	m := &fakeModel{responses: []model.ChatResponse{
		{Message: model.Message{Role: "assistant", Content: "hello there"}},
	}}
	a := &Agent{Model: m, Tools: tools.NewRegistry()}

	history := []model.Message{{Role: "user", Content: "hi"}}
	reply, err := a.Run(context.Background(), &history)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if reply != "hello there" {
		t.Fatalf("Run: got %q, want %q", reply, "hello there")
	}
	if len(history) != 2 {
		t.Fatalf("Run: history has %d messages, want 2", len(history))
	}
}

func TestRunExecutesToolThenAnswers(t *testing.T) {
	registry := tools.NewRegistry()
	if err := registry.Register(echoTool{name: "read_file"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	m := &fakeModel{responses: []model.ChatResponse{
		{Message: model.Message{
			Role: "assistant",
			ToolCalls: []model.ToolCall{
				{ID: "call_1", Name: "read_file", Arguments: json.RawMessage(`{"path":"go.mod"}`)},
			},
		}},
		{Message: model.Message{Role: "assistant", Content: "the file says X"}},
	}}

	var gotCallName string
	var gotResult tools.ToolResult
	a := &Agent{
		Model: m,
		Tools: registry,
		OnToolCall: func(name string, _ json.RawMessage) {
			gotCallName = name
		},
		OnToolResult: func(_ string, result tools.ToolResult) {
			gotResult = result
		},
	}

	history := []model.Message{{Role: "user", Content: "what does go.mod say?"}}
	reply, err := a.Run(context.Background(), &history)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if reply != "the file says X" {
		t.Fatalf("Run: got %q, want %q", reply, "the file says X")
	}
	if gotCallName != "read_file" {
		t.Fatalf("OnToolCall: got name %q, want read_file", gotCallName)
	}
	if gotResult.Content != `echo:{"path":"go.mod"}` {
		t.Fatalf("OnToolResult: got %q", gotResult.Content)
	}

	// history: user, assistant(tool_call), tool(result), assistant(final)
	if len(history) != 4 {
		t.Fatalf("Run: history has %d messages, want 4: %+v", len(history), history)
	}
	if history[2].Role != "tool" {
		t.Fatalf("Run: history[2].Role = %q, want tool", history[2].Role)
	}
}

func TestRunUnknownToolReportsError(t *testing.T) {
	m := &fakeModel{responses: []model.ChatResponse{
		{Message: model.Message{
			Role: "assistant",
			ToolCalls: []model.ToolCall{
				{Name: "does_not_exist", Arguments: json.RawMessage(`{}`)},
			},
		}},
		{Message: model.Message{Role: "assistant", Content: "done"}},
	}}
	a := &Agent{Model: m, Tools: tools.NewRegistry()}

	history := []model.Message{{Role: "user", Content: "hi"}}
	if _, err := a.Run(context.Background(), &history); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(history[2].Content, "unknown tool") {
		t.Fatalf("expected tool-result message to report unknown tool, got %q", history[2].Content)
	}
}

func TestRunStopsAtMaxTurns(t *testing.T) {
	// A model that always requests a tool call, never finishing.
	loop := model.ChatResponse{Message: model.Message{
		Role:      "assistant",
		ToolCalls: []model.ToolCall{{Name: "noop", Arguments: json.RawMessage(`{}`)}},
	}}
	m := &fakeModel{responses: []model.ChatResponse{loop, loop, loop, loop, loop}}

	registry := tools.NewRegistry()
	if err := registry.Register(echoTool{name: "noop"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	a := &Agent{Model: m, Tools: registry, MaxTurns: 3}
	history := []model.Message{{Role: "user", Content: "hi"}}
	if _, err := a.Run(context.Background(), &history); err == nil {
		t.Fatal("Run: expected error when max turns exceeded")
	}
}
