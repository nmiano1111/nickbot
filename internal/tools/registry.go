package tools

import (
	"fmt"
	"sort"
)

// Registry holds the tools available to the agent, keyed by name. The
// agent loop should look tools up by name and dispatch generically —
// it should never switch on tool name itself.
type Registry struct {
	tools map[string]Tool
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register adds t to the registry. It errors if a tool with the same name
// is already registered.
func (r *Registry) Register(t Tool) error {
	if _, exists := r.tools[t.Name()]; exists {
		return fmt.Errorf("tool %q already registered", t.Name())
	}
	r.tools[t.Name()] = t
	return nil
}

// Get looks up a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools, sorted by name for deterministic
// ordering (matters once this feeds a model's tool-definitions list).
func (r *Registry) List() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}
