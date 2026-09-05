package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// maxGitDiffBytes bounds how much diff output a single call returns.
const maxGitDiffBytes = 32 * 1024

// GitDiffTool shows a read-only git diff: the working tree by default,
// staged changes, or a diff scoped to one file.
type GitDiffTool struct {
	Root string
}

func NewGitDiffTool(root string) *GitDiffTool {
	return &GitDiffTool{Root: root}
}

func (t *GitDiffTool) Name() string { return "git_diff" }

func (t *GitDiffTool) Description() string {
	return "Show the working tree or staged git diff, optionally scoped to one file."
}

func (t *GitDiffTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "workspace-relative file to scope the diff to (default: whole repo)",
			},
			"staged": map[string]any{
				"type":        "boolean",
				"description": "show staged changes instead of the working tree diff",
			},
		},
	}
}

type gitDiffArgs struct {
	Path   string `json:"path"`
	Staged bool   `json:"staged"`
}

func (t *GitDiffTool) Execute(ctx context.Context, args json.RawMessage) (ToolResult, error) {
	var a gitDiffArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &a); err != nil {
			return ToolResult{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
		}
	}

	argv := []string{"diff"}
	if a.Staged {
		argv = append(argv, "--cached")
	}
	if a.Path != "" {
		full, err := resolveInRoot(t.Root, a.Path)
		if err != nil {
			return ToolResult{Content: err.Error(), IsError: true}, nil
		}
		argv = append(argv, "--", full)
	}

	cmd := exec.CommandContext(ctx, "git", argv...)
	cmd.Dir = t.Root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return ToolResult{Content: fmt.Sprintf("git diff failed: %v: %s", err, strings.TrimSpace(stderr.String())), IsError: true}, nil
	}

	out := stdout.String()
	if out == "" {
		return ToolResult{Content: "(no changes)"}, nil
	}
	if len(out) > maxGitDiffBytes {
		out = out[:maxGitDiffBytes] + fmt.Sprintf("\n\n[OUTPUT TRUNCATED: showing first %d bytes]", maxGitDiffBytes)
	}
	return ToolResult{Content: out}, nil
}
