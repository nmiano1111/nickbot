// Package repl implements the interactive chat loop.
package repl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"nickbot/internal/model"
)

const (
	dim   = "\x1b[2;3m"
	reset = "\x1b[0m"
)

// REPL runs an interactive chat session against a model.
type REPL struct {
	model   model.Model
	label   string
	system  string
	history []model.Message
}

// New returns a REPL ready to Run. label is a display string describing the
// model/backend, shown in the startup banner. system, if non-empty, is sent
// as the system prompt at the start of every conversation.
func New(m model.Model, label, system string) *REPL {
	r := &REPL{model: m, label: label, system: system}
	r.resetHistory()
	return r
}

func (r *REPL) resetHistory() {
	r.history = r.history[:0]
	if r.system != "" {
		r.history = append(r.history, model.Message{Role: "system", Content: r.system})
	}
}

// Run reads user input from in, prints replies (and streamed thinking
// tokens) to out, and blocks until in is exhausted, ctx is canceled, or
// /exit is entered.
func (r *REPL) Run(ctx context.Context, in io.Reader, out io.Writer) error {
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

		r.history = append(r.history, model.Message{Role: "user", Content: input})

		reply, err := r.exchange(ctx, out)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			r.history = r.history[:len(r.history)-1]
			continue
		}
		r.history = append(r.history, model.Message{Role: "assistant", Content: reply})
	}
}

// exchange sends the current history to the model and streams the reply to
// out, dimming thinking tokens and switching to normal style for content.
func (r *REPL) exchange(ctx context.Context, out io.Writer) (string, error) {
	fmt.Fprint(out, "\n")
	var thinking, answering bool

	onThinking := func(tok string) {
		if !thinking {
			fmt.Fprint(out, dim)
			thinking = true
		}
		fmt.Fprint(out, tok)
	}
	onContent := func(tok string) {
		if thinking && !answering {
			fmt.Fprint(out, reset+"\n\n")
		}
		answering = true
		fmt.Fprint(out, tok)
	}

	resp, err := r.model.Chat(ctx, model.ChatRequest{
		Messages:   r.history,
		OnThinking: onThinking,
		OnContent:  onContent,
	})
	if thinking && !answering {
		fmt.Fprint(out, reset)
	}
	fmt.Fprintln(out)
	return resp.Message.Content, err
}
