package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestGrepFindsMatches(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "service.go"), "type SubscriptionService struct{}\nfunc unrelated() {}\n")
	mustMkdir(t, filepath.Join(dir, "node_modules"))
	mustWriteFile(t, filepath.Join(dir, "node_modules", "junk.go"), "SubscriptionService should be excluded\n")

	tool := NewGrepTool(dir)
	args, _ := json.Marshal(map[string]any{"pattern": "SubscriptionService"})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.IsError {
		t.Fatalf("Execute: unexpected error result: %s", res.Content)
	}
	if !strings.Contains(res.Content, "service.go:1:") {
		t.Errorf("expected match in service.go, got:\n%s", res.Content)
	}
	if strings.Contains(res.Content, "node_modules") {
		t.Errorf("expected node_modules to be excluded, got:\n%s", res.Content)
	}
}

func TestGrepNoMatches(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "a.go"), "nothing interesting here\n")

	tool := NewGrepTool(dir)
	args, _ := json.Marshal(map[string]any{"pattern": "NoSuchThing"})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Content != "no matches" {
		t.Errorf("expected 'no matches', got:\n%s", res.Content)
	}
}

func TestGrepInvalidPattern(t *testing.T) {
	dir := t.TempDir()
	tool := NewGrepTool(dir)
	args, _ := json.Marshal(map[string]any{"pattern": "("})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected error result for invalid regex")
	}
}
