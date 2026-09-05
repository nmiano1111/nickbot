package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, dir, name string, lines int) {
	t.Helper()
	var sb strings.Builder
	for i := 1; i <= lines; i++ {
		sb.WriteString("line " + strconv.Itoa(i) + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("writeTestFile: %v", err)
	}
}

func TestReadFileBasicRange(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "foo.txt", 10)

	tool := NewReadFileTool(dir)
	args, _ := json.Marshal(map[string]any{"path": "foo.txt", "start_line": 3, "end_line": 5})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if res.IsError {
		t.Fatalf("Execute: unexpected error result: %s", res.Content)
	}
	for _, want := range []string{"3: line 3", "4: line 4", "5: line 5"} {
		if !strings.Contains(res.Content, want) {
			t.Errorf("Execute: content missing %q, got:\n%s", want, res.Content)
		}
	}
	if strings.Contains(res.Content, "line 6") {
		t.Errorf("Execute: content should not include line 6, got:\n%s", res.Content)
	}
}

func TestReadFileTruncatesLongFiles(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "big.txt", maxReadFileLines+50)

	tool := NewReadFileTool(dir)
	args, _ := json.Marshal(map[string]any{"path": "big.txt"})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(res.Content, "OUTPUT TRUNCATED") {
		t.Errorf("Execute: expected truncation notice, got:\n%s", res.Content)
	}
	if strings.Contains(res.Content, "line "+strconv.Itoa(maxReadFileLines+1)) {
		t.Errorf("Execute: should not include lines past the cap")
	}
}

func TestReadFileRejectsPathEscape(t *testing.T) {
	dir := t.TempDir()
	tool := NewReadFileTool(dir)
	args, _ := json.Marshal(map[string]any{"path": "../../etc/passwd"})
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !res.IsError {
		t.Fatal("Execute: expected error result for path escaping workspace root")
	}
}
