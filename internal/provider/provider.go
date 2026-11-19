// Package provider generates commit messages with an AI provider. The prompt
// and output parsing are shared; each provider only sends text to its model.
package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// Provider sends a prompt to an AI model and returns its raw text reply.
type Provider interface {
	Name() string
	DefaultModel() string
	// Complete sends system + input to model and returns the raw reply.
	Complete(ctx context.Context, model, system, input string) (string, error)
}

// Default is the provider used when aicm.provider and -P are unset.
const Default = "claude"

var registry = map[string]Provider{
	"claude": Claude{},
}

// Get returns the provider registered under name.
func Get(name string) (Provider, error) {
	if p, ok := registry[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("unknown provider %q (available: %s)", name, strings.Join(Names(), ", "))
}

// Names lists the registered providers, sorted.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// separator splits the two variants returned when a ticket is present.
const separator = "==="

// Request describes one generation.
type Request struct {
	Model string // empty means the provider's default
	Diff  string
	Stat  string
	Rules string // free-text style instructions from aicm.rules
	// Variants asks for two messages: a Conventional Commit and a plain
	// subject without a type prefix. Used when a ticket will be prepended.
	Variants bool
}

const basePrompt = `You write git commit messages from a staged diff.

Output ONLY the commit message: no preamble, no explanation, no code fences.

Use Conventional Commits: "<type>(<optional scope>): <subject>".
Types: feat, fix, chore, refactor, docs, test, perf, build, ci, style.
- Subject: imperative mood, lowercase start, no trailing period, at most 72 characters.
- Add a body (after one blank line) only when the change needs explaining; wrap at 72 characters and say why, not how.
- Never mention ticket or issue IDs; they are added separately.`

const variantsPrompt = `

Return TWO alternative messages separated by a line containing only "===":
1. A Conventional Commit as described above.
2. The same message WITHOUT the "<type>(<scope>): " prefix, starting with a capitalised imperative verb.`

// SystemPrompt builds the system prompt for req.
func SystemPrompt(req Request) string {
	p := basePrompt
	if req.Variants {
		p += variantsPrompt
	}
	if r := strings.TrimSpace(req.Rules); r != "" {
		p += "\n\nAdditional rules from the user (these take priority):\n" + r
	}
	return p
}

// Generate asks p for a message, or two when req.Variants is set.
func Generate(ctx context.Context, p Provider, req Request) ([]string, error) {
	model := req.Model
	if model == "" {
		model = p.DefaultModel()
	}
	input := "Files changed:\n" + req.Stat + "\nDiff:\n" + req.Diff
	out, err := p.Complete(ctx, model, SystemPrompt(req), input)
	if err != nil {
		return nil, err
	}

	msgs := Parse(out)
	if len(msgs) == 0 {
		return nil, fmt.Errorf("%s returned an empty message", p.Name())
	}
	if !req.Variants {
		msgs = msgs[:1]
	}
	return msgs, nil
}

// Parse cleans the model's output and splits it into variants.
func Parse(out string) []string {
	var msgs []string
	for _, part := range splitOnSeparator(out) {
		if m := clean(part); m != "" {
			msgs = append(msgs, m)
		}
	}
	return msgs
}

func splitOnSeparator(s string) []string {
	var parts []string
	var cur []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) == separator {
			parts = append(parts, strings.Join(cur, "\n"))
			cur = nil
			continue
		}
		cur = append(cur, line)
	}
	return append(parts, strings.Join(cur, "\n"))
}

// clean strips code fences, list numbering and surrounding whitespace.
func clean(s string) string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			continue
		}
		lines = append(lines, strings.TrimRight(line, " \t\r"))
	}
	s = strings.TrimSpace(strings.Join(lines, "\n"))
	for _, prefix := range []string{"1. ", "2. "} {
		s = strings.TrimPrefix(s, prefix)
	}
	return s
}
