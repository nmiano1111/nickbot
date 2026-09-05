// Package repl implements the interactive chat loop.
package repl

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"nickbot/internal/agent"
	"nickbot/internal/model"
	"nickbot/internal/tools"
)

const (
	dim   = "\x1b[2;3m"
	reset = "\x1b[0m"
)

// REPL runs an interactive chat session against an agent.
type REPL struct {
	agent   *agent.Agent
	label   string
	system  string
	history []model.Message

	out io.Writer
	// per-turn streaming state, reset by onTurnStart.
	turnThinking  bool
	turnAnswering bool
}

// New returns a REPL ready to Run. It wires itself up as a's streaming and
// tool-event callbacks. label is a display string describing the
// model/backend, shown in the startup banner. system, if non-empty, is
// sent as the system prompt at the start of every conversation.
func New(a *agent.Agent, label, system string) *REPL {
	r := &REPL{agent: a, label: label, system: system}
	a.OnTurnStart = r.onTurnStart
	a.OnThinking = r.onThinking
	a.OnContent = r.onContent
	a.OnToolCall = r.onToolCall
	a.OnToolResult = r.onToolResult
	r.resetHistory()
	return r
}

func (r *REPL) resetHistory() {
	r.history = r.history[:0]
	if r.system != "" {
		r.history = append(r.history, model.Message{Role: "system", Content: r.system})
	}
}

// Run reads user input from in, prints replies (thinking tokens, tool
// calls, and the final answer) to out, and blocks until in is exhausted,
// ctx is canceled, or /exit is entered.
func (r *REPL) Run(ctx context.Context, in io.Reader, out io.Writer) error {
	r.out = out
	fmt.Fprintf(out, "nickbot — chatting with %s\n", r.label)
	fmt.Fprintln(out, "Type your message and press Enter. Ctrl+D or /exit to quit, /reset to clear history.")

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for {
		fmt.Fprint(out, "\n> ")
		if !scanner.Scan() {
			fmt.Fprintln(out)
			return scanner.Err()
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		switch input {
		case "/exit", "/quit":
			return nil
		case "/reset":
			r.resetHistory()
			fmt.Fprintln(out, "(history cleared)")
			continue
		}

		beforeLen := len(r.history)
		r.history = append(r.history, model.Message{Role: "user", Content: input})

		_, err := r.agent.Run(ctx, &r.history)
		r.closeThinkingIfBare()
		fmt.Fprintln(out)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			r.history = r.history[:beforeLen]
		}
	}
}

func (r *REPL) onTurnStart() {
	r.turnThinking = false
	r.turnAnswering = false
}

func (r *REPL) onThinking(tok string) {
	if !r.turnThinking {
		fmt.Fprint(r.out, "\n"+dim)
		r.turnThinking = true
	}
	fmt.Fprint(r.out, tok)
}

func (r *REPL) onContent(tok string) {
	if r.turnThinking && !r.turnAnswering {
		fmt.Fprint(r.out, reset+"\n\n")
	} else if !r.turnAnswering {
		fmt.Fprint(r.out, "\n")
	}
	r.turnAnswering = true
	fmt.Fprint(r.out, tok)
}

// closeThinkingIfBare resets the dim style if the current turn streamed
// thinking tokens but never followed up with content (e.g. it went
// straight to a tool call, or the loop ended mid-turn).
func (r *REPL) closeThinkingIfBare() {
	if r.turnThinking && !r.turnAnswering {
		fmt.Fprint(r.out, reset)
	}
}

func (r *REPL) onToolCall(name string, args json.RawMessage) {
	r.closeThinkingIfBare()
	fmt.Fprintf(r.out, "\n● %s(%s)\n", name, formatArgs(args))
}

func (r *REPL) onToolResult(_ string, result tools.ToolResult) {
	if result.IsError {
		fmt.Fprintf(r.out, "  ⚠ %s\n", result.Content)
	}
}

// formatArgs renders a tool call's JSON arguments as compact "k=v, k2=v2"
// text for the transcript, falling back to the raw JSON if it isn't a
// simple object.
func formatArgs(raw json.RawMessage) string {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil || len(m) == 0 {
		return strings.TrimSpace(string(raw))
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=%v", k, m[k])
	}
	return strings.Join(parts, ", ")
}
