package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// maxReadFileLines bounds how many lines a single read_file call returns,
// so the model can't accidentally pull an entire giant file into context.
const maxReadFileLines = 500

// ReadFileTool reads a bounded range of lines from a file inside the
// workspace rooted at Root.
type ReadFileTool struct {
	Root string
}

func NewReadFileTool(root string) *ReadFileTool {
	return &ReadFileTool{Root: root}
}

func (t *ReadFileTool) Name() string { return "read_file" }

func (t *ReadFileTool) Description() string {
	return "Read a bounded range of lines from a file inside the project workspace."
}

func (t *ReadFileTool) Schema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"path"},
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "workspace-relative file path",
			},
			"start_line": map[string]any{
				"type":        "integer",
				"description": "1-indexed start line (default 1)",
			},
			"end_line": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("1-indexed end line, inclusive (default: start_line + %d)", maxReadFileLines-1),
			},
		},
	}
}

type readFileArgs struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

func (t *ReadFileTool) Execute(_ context.Context, args json.RawMessage) (ToolResult, error) {
	var a readFileArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}
	if a.Path == "" {
		return ToolResult{Content: "path is required", IsError: true}, nil
	}

	full, err := resolveInRoot(t.Root, a.Path)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	if a.StartLine <= 0 {
		a.StartLine = 1
	}
	maxEnd := a.StartLine + maxReadFileLines - 1
	capped := a.EndLine <= 0 || a.EndLine > maxEnd
	if capped {
		a.EndLine = maxEnd
	}

	f, err := os.Open(full)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("cannot open %s: %v", a.Path, err), IsError: true}, nil
	}
	defer f.Close()

	var out strings.Builder
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0
	moreAfter := false
	for scanner.Scan() {
		lineNum++
		if lineNum < a.StartLine {
			continue
		}
		if lineNum > a.EndLine {
			moreAfter = true
			break
		}
		fmt.Fprintf(&out, "%5d: %s\n", lineNum, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return ToolResult{Content: fmt.Sprintf("error reading %s: %v", a.Path, err), IsError: true}, nil
	}

	if out.Len() == 0 {
		return ToolResult{Content: fmt.Sprintf("no lines in range %d-%d", a.StartLine, a.EndLine)}, nil
	}
	if moreAfter {
		fmt.Fprintf(&out, "\n[OUTPUT TRUNCATED: showing lines %d-%d; request a further range to continue]\n", a.StartLine, a.EndLine)
	}

	return ToolResult{Content: out.String()}, nil
}
