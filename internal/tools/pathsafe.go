package tools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// resolveInRoot resolves rel against root and rejects any path that
// escapes root.
//
// This is a simple lexical containment check: it cleans and joins the
// paths and verifies the result still sits under root. It does not resolve
// symlinks, so a symlink inside root pointing outside it would slip
// through. That hardening belongs to the dedicated Workspace boundary
// (a later milestone) and matters most once write or shell-execution tools
// exist; for today's read-only tools it is an acceptable interim guard.
func resolveInRoot(root, rel string) (string, error) {
	root = filepath.Clean(root)
	full := filepath.Clean(filepath.Join(root, rel))
	if full != root && !strings.HasPrefix(full, root+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes workspace root", rel)
	}
	return full, nil
}
