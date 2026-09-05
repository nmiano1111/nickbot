package tools

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestGitDiffShowsWorkingTreeChange(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()

	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")

	mustWriteFile(t, filepath.Join(dir, "foo.txt"), "hello\n")
	runGit(t, dir, "add", "foo.txt")
	runGit(t, dir, "commit", "-q", "-m", "initial")

	mustWriteFile(t, filepath.Join(dir, "foo.txt"), "hello world\n")

	tool := NewGitDiffTool(dir)
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.IsError {
		t.Fatalf("Execute: unexpected error result: %s", res.Content)
	}
	if !strings.Contains(res.Content, "hello world") {
		t.Errorf("expected diff to mention the change, got:\n%s", res.Content)
	}
}

func TestGitDiffNoChanges(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()

	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	mustWriteFile(t, filepath.Join(dir, "foo.txt"), "hello\n")
	runGit(t, dir, "add", "foo.txt")
	runGit(t, dir, "commit", "-q", "-m", "initial")

	tool := NewGitDiffTool(dir)
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.Content != "(no changes)" {
		t.Errorf("expected '(no changes)', got:\n%s", res.Content)
	}
}
