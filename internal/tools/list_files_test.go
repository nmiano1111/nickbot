package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListFilesRespectsDepthAndExcludes(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "a", "b"))
	mustMkdir(t, filepath.Join(dir, "node_modules"))
	mustWriteFile(t, filepath.Join(dir, "a", "top.go"), "x")
	mustWriteFile(t, filepath.Join(dir, "a", "b", "deep.go"), "x")
	mustWriteFile(t, filepath.Join(dir, "node_modules", "junk.js"), "x")

	tool := NewListFilesTool(dir)
	args, _ := json.Marshal(map[string]any{"depth": 2})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(res.Content, "a/") {
		t.Errorf("expected top-level dir a/ in output, got:\n%s", res.Content)
	}
	if !strings.Contains(res.Content, "a/top.go") {
		t.Errorf("expected a/top.go in output, got:\n%s", res.Content)
	}
	if strings.Contains(res.Content, "deep.go") {
		t.Errorf("depth 2 should not descend into a/b, got:\n%s", res.Content)
	}
	if strings.Contains(res.Content, "node_modules") {
		t.Errorf("node_modules should be excluded, got:\n%s", res.Content)
	}
}

func TestListFilesRejectsPathEscape(t *testing.T) {
	dir := t.TempDir()
	tool := NewListFilesTool(dir)
	args, _ := json.Marshal(map[string]any{"path": "../"})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.IsError {
		t.Fatal("Execute: expected error result for path escaping workspace root")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
