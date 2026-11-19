package provider

import (
	"slices"
	"testing"
)

func TestClaudeArgs(t *testing.T) {
	args := ClaudeArgs("sonnet", "sys")
	if slices.Contains(args, "--bare") {
		t.Error("--bare breaks subscription auth and must not be used")
	}
	if i := slices.Index(args, "--model"); i < 0 || args[i+1] != "sonnet" {
		t.Errorf("model not passed: %q", args)
	}
	if i := slices.Index(args, "--system-prompt"); i < 0 || args[i+1] != "sys" {
		t.Errorf("system prompt not passed: %q", args)
	}
}
