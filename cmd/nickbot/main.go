// Command nickbot is a REPL for chatting with a local model.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"nickbot/internal/model"
	"nickbot/internal/repl"
)

func main() {
	host := flag.String("host", "http://localhost:11434", "Ollama server base URL")
	modelName := flag.String("model", "qwen3.5:9b", "model name to chat with")
	system := flag.String("system", "", "optional system prompt")
	flag.Parse()

	m := model.NewOllama(*host, *modelName)
	label := fmt.Sprintf("%s at %s", *modelName, *host)
	r := repl.New(m, label, *system)

	if err := r.Run(context.Background(), os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
