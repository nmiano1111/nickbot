package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// maxGrepResults bounds how many matches a single grep call returns. Do not
// automatically return entire matching files — the model can request
// specific line ranges via read_file afterward.
const maxGrepResults = 50

// grepFallbackHardCap is a safety valve on the pure-Go fallback search so a
// pathological pattern/repo can't scan forever; formatMatches trims further
// for display.
const grepFallbackHardCap = 5000

// GrepTool searches for a regex pattern across files under a
// workspace-relative path. It shells out to ripgrep when available and
// falls back to a pure-Go recursive search otherwise.
type GrepTool struct {
	Root string
}

func NewGrepTool(root string) *GrepTool {
	return &GrepTool{Root: root}
}

func (t *GrepTool) Name() string { return "grep" }

func (t *GrepTool) Description() string {
	return "Search for a regex pattern across files under a workspace-relative path, returning compact path:line:text matches."
}

func (t *GrepTool) Schema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"pattern"},
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "regular expression to search for",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "workspace-relative path to search (default: workspace root)",
			},
		},
	}
}

type grepArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

func (t *GrepTool) Execute(ctx context.Context, args json.RawMessage) (ToolResult, error) {
	var a grepArgs
	if err := json.Unmarshal(args, &a); err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid arguments: %v", err), IsError: true}, nil
	}
	if a.Pattern == "" {
		return ToolResult{Content: "pattern is required", IsError: true}, nil
	}

	full, err := resolveInRoot(t.Root, a.Path)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	if _, err := exec.LookPath("rg"); err == nil {
		return t.execRipgrep(ctx, full, a.Pattern)
	}
	return t.execFallback(a.Pattern, full)
}

func (t *GrepTool) execRipgrep(ctx context.Context, dir, pattern string) (ToolResult, error) {
	cmd := exec.CommandContext(ctx, "rg", "--line-number", "--no-heading", "--color=never", "--", pattern, ".")
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return ToolResult{Content: "no matches"}, nil
		}
		return ToolResult{Content: fmt.Sprintf("rg failed: %v: %s", err, strings.TrimSpace(stderr.String())), IsError: true}, nil
	}

	trimmed := strings.TrimRight(stdout.String(), "\n")
	if trimmed == "" {
		return ToolResult{Content: "no matches"}, nil
	}
	return ToolResult{Content: formatMatches(strings.Split(trimmed, "\n"), maxGrepResults)}, nil
}

func (t *GrepTool) execFallback(pattern, dir string) (ToolResult, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("invalid pattern: %v", err), IsError: true}, nil
	}

	var results []string
	walkErr := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if listFilesExcludes[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if len(results) >= grepFallbackHardCap {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		rel, _ := filepath.Rel(dir, path)
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			if re.MatchString(scanner.Text()) {
				results = append(results, fmt.Sprintf("%s:%d:%s", rel, lineNum, scanner.Text()))
				if len(results) >= grepFallbackHardCap {
					break
				}
			}
		}
		return nil
	})
	if walkErr != nil {
		return ToolResult{Content: fmt.Sprintf("search failed: %v", walkErr), IsError: true}, nil
	}

	sort.Strings(results)
	if len(results) == 0 {
		return ToolResult{Content: "no matches"}, nil
	}
	return ToolResult{Content: formatMatches(results, maxGrepResults)}, nil
}

func formatMatches(lines []string, max int) string {
	if len(lines) <= max {
		return strings.Join(lines, "\n")
	}
	return fmt.Sprintf("%s\n\n[OUTPUT TRUNCATED: showing first %d matches]", strings.Join(lines[:max], "\n"), max)
}
