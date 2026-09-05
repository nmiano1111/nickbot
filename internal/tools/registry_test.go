package tools

import (
	"context"
	"encoding/json"
	"testing"
)

type fakeTool struct {
	name string
}

func (f fakeTool) Name() string        { return f.name }
func (f fakeTool) Description() string { return "a fake tool for tests" }
func (f fakeTool) Schema() map[string]any {
	return map[string]any{"type": "object"}
}
func (f fakeTool) Execute(ctx context.Context, args json.RawMessage) (ToolResult, error) {
	return ToolResult{Content: "ok"}, nil
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(fakeTool{name: "read_file"}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, ok := r.Get("read_file")
	if !ok {
		t.Fatal("Get: expected tool to be found")
	}
	if got.Name() != "read_file" {
		t.Fatalf("Get: got tool named %q, want read_file", got.Name())
	}

	if _, ok := r.Get("missing"); ok {
		t.Fatal("Get: expected missing tool to not be found")
	}
}

func TestRegistryDuplicateNameErrors(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(fakeTool{name: "grep"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if err := r.Register(fakeTool{name: "grep"}); err == nil {
		t.Fatal("Register: expected error registering duplicate tool name")
	}
}

func TestRegistryListSortedByName(t *testing.T) {
	r := NewRegistry()
	for _, name := range []string{"grep", "read_file", "list_files"} {
		if err := r.Register(fakeTool{name: name}); err != nil {
			t.Fatalf("Register(%q): %v", name, err)
		}
	}

	got := r.List()
	want := []string{"grep", "list_files", "read_file"}
	if len(got) != len(want) {
		t.Fatalf("List: got %d tools, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i].Name() != name {
			t.Fatalf("List[%d]: got %q, want %q", i, got[i].Name(), name)
		}
	}
}
