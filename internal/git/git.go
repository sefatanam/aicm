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

// Config returns every aicm.* git config value, keyed by lowercase name
// without the "aicm." prefix. Git already applies repo-over-global precedence.
func Config() map[string]string {
	cfg := map[string]string{}
	out, err := run("config", "--get-regexp", `^aicm\.`)
	if err != nil {
		return cfg // exit 1 just means no keys are set
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		key, val, _ := strings.Cut(line, " ")
		cfg[strings.ToLower(strings.TrimPrefix(key, "aicm."))] = val
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
