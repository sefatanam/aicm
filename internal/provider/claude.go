package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Claude runs the Claude Code CLI in print mode, which authenticates with the
// user's subscription login, so no API key is needed.
type Claude struct{}

func (Claude) Name() string         { return "claude" }
func (Claude) DefaultModel() string { return "haiku" }

// ClaudeArgs returns the claude CLI arguments. It deliberately avoids --bare,
// which only accepts ANTHROPIC_API_KEY and breaks subscription auth.
func ClaudeArgs(model, system string) []string {
	return []string{
		"-p",
		"--model", model,
		"--tools", "",
		"--no-session-persistence",
		"--setting-sources", "",
		"--strict-mcp-config",
		"--system-prompt", system,
	}
}

func (Claude) Complete(ctx context.Context, model, system, input string) (string, error) {
	if _, err := exec.LookPath("claude"); err != nil {
		return "", errors.New("claude not found on PATH; install Claude Code (https://claude.com/claude-code) and run `claude` once to log in")
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "claude", ClaudeArgs(model, system)...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("claude failed: %s", msg)
	}
	return stdout.String(), nil
}
