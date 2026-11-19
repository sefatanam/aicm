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

// gitError is a failed git command. It wraps the exec error, so callers can
// tell an exit status apart from other failures.
type gitError struct {
	cmd string // the git subcommand, e.g. "config"
	msg string // git's stderr, or the exec error when stderr is empty
	err error
}

func (e *gitError) Error() string { return fmt.Sprintf("git %s: %s", e.cmd, e.msg) }
func (e *gitError) Unwrap() error { return e.err }

func run(args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", &gitError{args[0], msg, err}
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
// (system, global, local), so later entries override earlier ones. Exit
// status 1 from git means no aicm keys are set; any other failure is an error.
func ConfigEntries() ([]Entry, error) {
	out, err := run("config", "-z", "--show-scope", "--show-origin", "--get-regexp", `^aicm\.`)
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseEntries(out), nil
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

// Lookup returns the entries for key, in precedence order.
func Lookup(entries []Entry, key string) []Entry {
	var out []Entry
	for _, e := range entries {
		if e.Key == key {
			out = append(out, e)
		}
	}
	return out
}

// Values maps each key to the value in effect. Repo settings override global
// ones because they come later in entries.
func Values(entries []Entry) map[string]string {
	cfg := map[string]string{}
	for _, e := range entries {
		cfg[e.Key] = e.Value
	}
	return cfg
}

// Config returns the aicm.* values in effect, keyed by lowercase name
// without the "aicm." prefix.
func Config() (map[string]string, error) {
	entries, err := ConfigEntries()
	if err != nil {
		return nil, err
	}
	return Values(entries), nil
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
