// Package ui handles terminal interaction: TTY detection, the spinner, the
// accept/edit/regenerate prompt and opening the user's editor.
package ui

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// IsTerminal reports whether f is attached to a terminal.
func IsTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// Spin draws a spinner with label on w until ctx is cancelled, then clears it.
func Spin(ctx context.Context, w io.Writer, label string) {
	frames := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
	t := time.NewTicker(80 * time.Millisecond)
	defer t.Stop()
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			fmt.Fprint(w, "\r\033[K")
			return
		case <-t.C:
			fmt.Fprintf(w, "\r%c %s", frames[i%len(frames)], label)
		}
	}
}

// Action is the user's choice at the prompt.
type Action int

const (
	Commit Action = iota
	Edit
	Regenerate
	Quit
)

// Prompter reads answers from in and writes prompts to out.
type Prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func NewPrompter(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: bufio.NewReader(in), out: out}
}

func (p *Prompter) readLine() string {
	line, _ := p.in.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(line))
}

// Pick shows numbered variants and returns the chosen index. Enter picks 1.
func (p *Prompter) Pick(variants []string) int {
	for i, v := range variants {
		fmt.Fprintf(p.out, "\n\033[1m[%d]\033[0m %s\n", i+1, indent(v))
	}
	for {
		fmt.Fprintf(p.out, "\nPick a message [1-%d] (default 1): ", len(variants))
		ans := p.readLine()
		if ans == "" {
			return 0
		}
		var n int
		if _, err := fmt.Sscan(ans, &n); err == nil && n >= 1 && n <= len(variants) {
			return n - 1
		}
	}
}

// Confirm shows msg and asks what to do with it.
func (p *Prompter) Confirm(msg string, showMsg bool) Action {
	if showMsg {
		fmt.Fprintf(p.out, "\n%s\n", indent(msg))
	}
	for {
		fmt.Fprint(p.out, "\nCommit? [y]es / [e]dit / [r]egenerate / [n]o: ")
		switch p.readLine() {
		case "", "y", "yes":
			return Commit
		case "e", "edit":
			return Edit
		case "r", "regenerate":
			return Regenerate
		case "n", "no", "q", "quit":
			return Quit
		}
	}
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(s, "\n", "\n    ")
}

// EditInEditor opens msg in $VISUAL / $EDITOR and returns the saved text.
func EditInEditor(msg string) (string, error) {
	f, err := os.CreateTemp("", "aicm-edit-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(msg + "\n"); err != nil {
		f.Close()
		return "", err
	}
	f.Close()

	editor := strings.Fields(editorCommand())
	cmd := exec.Command(editor[0], append(editor[1:], f.Name())...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor %q: %w", editor[0], err)
	}
	out, err := os.ReadFile(f.Name())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func editorCommand() string {
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if e := strings.TrimSpace(os.Getenv(env)); e != "" {
			return e
		}
	}
	if runtime.GOOS == "windows" {
		return "notepad"
	}
	return "vi"
}
