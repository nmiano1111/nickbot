// Command nickbot is a small coding agent REPL backed by a local model.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"nickbot/internal/agent"
	"nickbot/internal/model"
	"nickbot/internal/repl"
	"nickbot/internal/tools"
)

const defaultSystemPrompt = `You are a software engineering agent operating within a project.

You have tools for inspecting the codebase.

Before answering questions about the codebase, inspect relevant files
rather than guessing.

Prefer targeted searches and file ranges over reading large amounts
of irrelevant code.

Do not claim that a tool action succeeded unless its result confirms
success.

When you have enough evidence to answer the user's request, stop
calling tools and provide a concise explanation.`

func main() {
	host := flag.String("host", "http://localhost:11434", "Ollama server base URL")
	modelName := flag.String("model", "qwen3.5:9b", "model name to chat with")
	root := flag.String("root", ".", "workspace root that filesystem tools are confined to")
	system := flag.String("system", defaultSystemPrompt, "system prompt")
	maxTurns := flag.Int("max-turns", 8, "max model/tool round trips per user message")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolving workspace root: %v\n", err)
		os.Exit(1)
	}

	registry := tools.NewRegistry()
	for _, t := range []tools.Tool{
		tools.NewReadFileTool(absRoot),
		tools.NewListFilesTool(absRoot),
		tools.NewGrepTool(absRoot),
		tools.NewGitDiffTool(absRoot),
	} {
		if err := registry.Register(t); err != nil {
			fmt.Fprintf(os.Stderr, "registering tool: %v\n", err)
			os.Exit(1)
		}
	}

	m := model.NewOllama(*host, *modelName)
	a := &agent.Agent{Model: m, Tools: registry, MaxTurns: *maxTurns}

	label := fmt.Sprintf("%s at %s (workspace: %s)", *modelName, *host, absRoot)
	r := repl.New(a, label, *system)

	if err := r.Run(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
