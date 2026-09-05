package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// maxListEntries bounds how many entries a single list_files call returns,
// so the model can't accidentally pull an enormous directory tree into
// context.
const maxListEntries = 500

// listFilesExcludes are directories that are rarely useful for an agent to
// browse and can be noisy or huge. Configurability can come later.
var listFilesExcludes = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
}

// ListFilesTool lists files and directories under a workspace-relative
// path, up to a given depth.
type ListFilesTool struct {
	Root string
}

func NewListFilesTool(root string) *ListFilesTool {
	return &ListFilesTool{Root: root}
}

func (t *ListFilesTool) Name() string { return "list_files" }

func (t *ListFilesTool) Description() string {
	return "List files and directories under a workspace-relative path, up to a given depth."
}

func (t *ListFilesTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "workspace-relative directory (default: workspace root)",
			},
			"depth": map[string]any{
				"type":        "integer",
				"description": "how many directory levels to descend (default 2)",
			},
		},
	}
}

type listFilesArgs struct {
	Path  string `json:"path"`
	Depth int    `json:"depth"`
}

func (t *ListFilesTool) Execute(_ context.Context, args json.RawMessage) (ToolResult, error) {
	var a listFilesArgs
	if len(args) > 0 {
		if err := json.Unmarshal(args, &a); err != nil {
			return ToolResult{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
		}
	}
	if a.Depth <= 0 {
		a.Depth = 2
	}

	full, err := resolveInRoot(t.Root, a.Path)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	info, err := os.Stat(full)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("cannot stat %s: %v", a.Path, err), IsError: true}, nil
	}
	if !info.IsDir() {
		return ToolResult{Content: fmt.Sprintf("%s is not a directory", a.Path)}, nil
	}

	var entries []string
	truncated := false

	var walk func(dir, relPrefix string, depth int)
	walk = func(dir, relPrefix string, depth int) {
		if truncated {
			return
		}
		dirEntries, err := os.ReadDir(dir)
		if err != nil {
			entries = append(entries, fmt.Sprintf("%s [error: %v]", relPrefix, err))
			return
		}
		sort.Slice(dirEntries, func(i, j int) bool { return dirEntries[i].Name() < dirEntries[j].Name() })
		for _, e := range dirEntries {
			if listFilesExcludes[e.Name()] {
				continue
			}
			relPath := e.Name()
			if relPrefix != "" {
				relPath = relPrefix + "/" + e.Name()
			}
			label := relPath
			if e.IsDir() {
				label += "/"
			}
			entries = append(entries, label)
			if len(entries) >= maxListEntries {
				truncated = true
				return
			}
			if e.IsDir() && depth > 1 {
				walk(filepath.Join(dir, e.Name()), relPath, depth-1)
			}
		}
	}
	walk(full, "", a.Depth)

	out := strings.Join(entries, "\n")
	if truncated {
		out += fmt.Sprintf("\n\n[OUTPUT TRUNCATED: showing first %d entries]", maxListEntries)
	}
	if out == "" {
		out = "(empty directory)"
	}
	return ToolResult{Content: out}, nil
}
