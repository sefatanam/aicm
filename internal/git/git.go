// Package git wraps the git commands aicm needs.
package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func run(args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", args[0], msg)
	}
	return stdout.String(), nil
}

// EnsureRepo returns an error if the working directory is not inside a git repo.
func EnsureRepo() error {
	if _, err := run("rev-parse", "--git-dir"); err != nil {
		return errors.New("not a git repository")
	}
	return nil
}

// StageTracked stages modifications and deletions of tracked files (git add -u).
func StageTracked() error {
	_, err := run("add", "-u")
	return err
}

// Staged returns the staged diff and its --stat summary.
func Staged() (diff, stat string, err error) {
	if diff, err = run("diff", "--cached", "--no-color", "--no-ext-diff"); err != nil {
		return "", "", err
	}
	if stat, err = run("diff", "--cached", "--no-color", "--stat"); err != nil {
		return "", "", err
	}
	return diff, stat, nil
}

// Branch returns the current branch name, or "" when HEAD is detached.
func Branch() string {
	out, err := run("branch", "--show-current")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// Entry is one aicm.* setting as git reports it. A key can appear once per
// scope; the last entry for a key is the one in effect.
type Entry struct {
	Scope  string // system, global, local, worktree or command
	Origin string // the file it came from, e.g. /home/u/.gitconfig
	Key    string // lowercase name without the "aicm." prefix
	Value  string
}

// ConfigEntries returns every aicm.* setting in git's precedence order
// (system, global, local), so later entries override earlier ones.
func ConfigEntries() []Entry {
	out, err := run("config", "-z", "--show-scope", "--show-origin", "--get-regexp", `^aicm\.`)
	if err != nil {
		return nil // exit 1 just means no keys are set
	}
	return parseEntries(out)
}

// parseEntries reads `git config -z --show-scope --show-origin` output:
// scope, origin and "key\nvalue" fields, each terminated by NUL. -z keeps
// multi-line values intact.
func parseEntries(out string) []Entry {
	var entries []Entry
	f := strings.Split(out, "\x00")
	for i := 0; i+2 < len(f); i += 3 {
		key, val, _ := strings.Cut(f[i+2], "\n")
		entries = append(entries, Entry{
			Scope:  f[i],
			Origin: strings.TrimPrefix(f[i+1], "file:"),
			Key:    strings.ToLower(strings.TrimPrefix(key, "aicm.")),
			Value:  val,
		})
	}
	return entries
}

// Config returns the aicm.* values in effect, keyed by lowercase name
// without the "aicm." prefix. Repo settings override global ones.
func Config() map[string]string {
	cfg := map[string]string{}
	for _, e := range ConfigEntries() {
		cfg[e.Key] = e.Value
	}
	return cfg
}

// Commit runs git commit with msg, attached to the terminal so the user's
// hooks and git's own output are visible.
func Commit(msg string) error {
	f, err := os.CreateTemp("", "aicm-msg-*.txt")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(msg + "\n"); err != nil {
		f.Close()
		return err
	}
	f.Close()

	cmd := exec.Command("git", "commit", "-F", f.Name())
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stderr, os.Stderr
	return cmd.Run()
}
