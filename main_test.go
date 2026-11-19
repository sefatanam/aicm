package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeClaude prints the given output and records its arguments.
const fakeClaude = `#!/bin/sh
printf '%s\n' "$@" > "$CM_TEST_ARGS"
cat > /dev/null
printf '%s' "$CM_TEST_OUTPUT"
`

type env struct {
	t       *testing.T
	bin     string
	repo    string
	path    string
	argsLog string
}

func setup(t *testing.T) *env {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake claude is a shell script")
	}
	dir := t.TempDir()
	bin := filepath.Join(dir, "aicm")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	fakeDir := filepath.Join(dir, "fakebin")
	os.Mkdir(fakeDir, 0o755)
	if err := os.WriteFile(filepath.Join(fakeDir, "claude"), []byte(fakeClaude), 0o755); err != nil {
		t.Fatal(err)
	}

	e := &env{
		t:       t,
		bin:     bin,
		repo:    filepath.Join(dir, "repo"),
		path:    fakeDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		argsLog: filepath.Join(dir, "args.txt"),
	}
	os.Mkdir(e.repo, 0o755)
	e.git("init", "-q", "-b", "main")
	e.git("config", "user.email", "test@example.com")
	e.git("config", "user.name", "Test")
	e.git("commit", "-q", "--allow-empty", "-m", "init")
	return e
}

func (e *env) git(args ...string) string {
	e.t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = e.repo
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		e.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (e *env) stage(name, content string) {
	e.t.Helper()
	os.WriteFile(filepath.Join(e.repo, name), []byte(content), 0o644)
	e.git("add", name)
}

// aicm runs the binary with stdout captured, so it is never a terminal.
func (e *env) aicm(claudeOutput string, args ...string) (string, error) {
	cmd := exec.Command(e.bin, args...)
	cmd.Dir = e.repo
	cmd.Env = append(os.Environ(),
		"PATH="+e.path,
		"GIT_CONFIG_GLOBAL=/dev/null",
		"CM_TEST_OUTPUT="+claudeOutput,
		"CM_TEST_ARGS="+e.argsLog,
	)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func TestPipedPrintsWithoutCommitting(t *testing.T) {
	e := setup(t)
	e.stage("a.go", "package a\n")
	out, err := e.aicm("feat: add package a\n")
	if err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if out != "feat: add package a" {
		t.Errorf("output = %q", out)
	}
	if got := e.git("log", "--format=%s", "-1"); got != "init" {
		t.Errorf("should not commit, HEAD is %q", got)
	}
}

func TestYesCommits(t *testing.T) {
	e := setup(t)
	e.stage("a.go", "package a\n")
	if out, err := e.aicm("feat: add package a\n\nWhy it matters.", "-y"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if got := e.git("log", "--format=%B", "-1"); got != "feat: add package a\n\nWhy it matters." {
		t.Errorf("commit message = %q", got)
	}
}

func TestNothingStaged(t *testing.T) {
	e := setup(t)
	out, err := e.aicm("feat: x")
	if err == nil || !strings.Contains(out, "nothing staged") {
		t.Errorf("want nothing-staged error, got %v: %s", err, out)
	}
}

func TestAllStagesTracked(t *testing.T) {
	e := setup(t)
	e.stage("a.go", "package a\n")
	e.git("commit", "-q", "-m", "add a")
	os.WriteFile(filepath.Join(e.repo, "a.go"), []byte("package a\n\nvar X = 1\n"), 0o644)
	if out, err := e.aicm("feat: add X", "-a", "-y"); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if got := e.git("log", "--format=%s", "-1"); got != "feat: add X" {
		t.Errorf("HEAD = %q", got)
	}
}

func TestTicketFromBranchUsesTemplateAndFirstVariant(t *testing.T) {
	e := setup(t)
	e.git("checkout", "-q", "-b", "feature/AXE-123-login")
	e.stage("a.go", "package a\n")
	out, err := e.aicm("feat(auth): add login\n===\nAdd login\n")
	if err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	if out != "AXE-123: feat(auth): add login" {
		t.Errorf("output = %q", out)
	}

	e.git("config", "aicm.format", "[{ticket}] {message}")
	out, _ = e.aicm("feat(auth): add login\n===\nAdd login\n", "-t", "ONE-0001")
	if out != "[ONE-0001] feat(auth): add login" {
		t.Errorf("custom template output = %q", out)
	}
}

func TestModelPrecedence(t *testing.T) {
	e := setup(t)
	e.stage("a.go", "package a\n")
	model := func(args ...string) string {
		t.Helper()
		if out, err := e.aicm("feat: x", args...); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
		log, _ := os.ReadFile(e.argsLog)
		lines := strings.Split(string(log), "\n")
		for i, l := range lines {
			if l == "--model" {
				return lines[i+1]
			}
		}
		return ""
	}
	if got := model(); got != "haiku" {
		t.Errorf("default model = %q", got)
	}
	e.git("config", "aicm.model", "sonnet")
	if got := model(); got != "sonnet" {
		t.Errorf("config model = %q", got)
	}
	if got := model("-m", "opus"); got != "opus" {
		t.Errorf("flag model = %q", got)
	}
}

func TestProviderSelection(t *testing.T) {
	e := setup(t)
	e.stage("a.go", "package a\n")
	if out, err := e.aicm("feat: x", "-P", "claude"); err != nil || out != "feat: x" {
		t.Errorf("-P claude: %v: %s", err, out)
	}
	out, err := e.aicm("feat: x", "-P", "nope")
	if err == nil || !strings.Contains(out, `unknown provider "nope"`) || !strings.Contains(out, "claude") {
		t.Errorf("-P nope: want unknown-provider error, got %v: %s", err, out)
	}
	e.git("config", "aicm.provider", "nope")
	if out, err := e.aicm("feat: x"); err == nil || !strings.Contains(out, "unknown provider") {
		t.Errorf("aicm.provider=nope: want error, got %v: %s", err, out)
	}
}

func TestTemplateWithoutTicket(t *testing.T) {
	e := setup(t)
	e.stage("a.go", "package a\n")
	e.git("config", "aicm.format", "[web] {message}")
	out, err := e.aicm("feat(auth): add login\n===\nAdd login\n")
	if err != nil || out != "[web] feat(auth): add login" {
		t.Errorf("plain template on main: %v: %q", err, out)
	}
	if log, _ := os.ReadFile(e.argsLog); !strings.Contains(string(log), "TWO") {
		t.Error("an applied template should ask for two variants")
	}

	out, _ = e.aicm("feat(auth): add login\n===\nAdd login\n", "-t", "AXE-9")
	if !strings.Contains(out, "warning") || !strings.HasSuffix(out, "[web] feat(auth): add login") {
		t.Errorf("-t with a template lacking {ticket} should warn: %q", out)
	}

	e.git("config", "aicm.format", "[web]")
	if out, err := e.aicm("feat: x"); err == nil || !strings.Contains(out, "{message}") {
		t.Errorf("template without {message}: want error, got %v: %q", err, out)
	}
}

func TestTicketTemplateFallsBackWithoutTicket(t *testing.T) {
	e := setup(t)
	e.stage("a.go", "package a\n")
	e.git("config", "aicm.format", "{ticket} | {message}")
	out, err := e.aicm("feat(auth): add login\n===\nAdd login\n")
	if err != nil || out != "feat(auth): add login" {
		t.Errorf("no ticket: want plain conventional commit, got %v: %q", err, out)
	}
	if log, _ := os.ReadFile(e.argsLog); strings.Contains(string(log), "TWO") {
		t.Error("no template applied, so only one message should be requested")
	}

	e.git("checkout", "-q", "-b", "feature/AXE-123-login")
	if out, _ := e.aicm("feat(auth): add login\n===\nAdd login\n"); out != "AXE-123 | feat(auth): add login" {
		t.Errorf("ticket branch: %q", out)
	}
}

func TestStylePrompt(t *testing.T) {
	e := setup(t)
	e.stage("a.go", "package a\n")
	e.git("config", "aicm.prompt", "Use gitmoji.\nOne line only.")
	if out, err := e.aicm("✨ add login"); err != nil || out != "✨ add login" {
		t.Fatalf("%v: %q", err, out)
	}
	log, _ := os.ReadFile(e.argsLog)
	if !strings.Contains(string(log), "Use gitmoji.") || strings.Contains(string(log), "Conventional Commits") {
		t.Errorf("aicm.prompt should replace the default style; claude args:\n%s", log)
	}
}
