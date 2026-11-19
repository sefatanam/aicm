package provider

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name, in string
		want     []string
	}{
		{"single", "feat: add login\n", []string{"feat: add login"}},
		{"body", "fix: crash\n\nNil map on startup.\n", []string{"fix: crash\n\nNil map on startup."}},
		{"fenced", "```\nfeat: add login\n```", []string{"feat: add login"}},
		{"variants", "feat(auth): add login\n===\nAdd login\n", []string{"feat(auth): add login", "Add login"}},
		{"numbered variants", "1. feat: x\n===\n2. Add x", []string{"feat: x", "Add x"}},
		{"empty", "  \n", nil},
	}
	for _, c := range cases {
		if got := Parse(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Parse(%q) = %q, want %q", c.name, c.in, got, c.want)
		}
	}
}

func TestSystemPrompt(t *testing.T) {
	p := SystemPrompt(Request{Variants: true, Rules: "lowercase, no period"})
	if !strings.Contains(p, separator) || !strings.Contains(p, "lowercase, no period") {
		t.Errorf("prompt missing variants or rules:\n%s", p)
	}
	if strings.Contains(SystemPrompt(Request{}), "TWO") {
		t.Error("single-variant prompt should not ask for two messages")
	}
}

func TestGet(t *testing.T) {
	if p, err := Get(Default); err != nil || p.Name() != Default {
		t.Fatalf("Get(%q) = %v, %v", Default, p, err)
	}
	_, err := Get("nope")
	if err == nil || !strings.Contains(err.Error(), "claude") {
		t.Errorf("unknown provider error should list available ones, got %v", err)
	}
}

type fake struct{ model, reply string }

func (*fake) Name() string         { return "fake" }
func (*fake) DefaultModel() string { return "small" }
func (f *fake) Complete(_ context.Context, model, _, _ string) (string, error) {
	f.model = model
	return f.reply, nil
}

func TestGenerate(t *testing.T) {
	f := &fake{reply: "feat: x\n===\nAdd x"}
	msgs, err := Generate(context.Background(), f, Request{})
	if err != nil || !reflect.DeepEqual(msgs, []string{"feat: x"}) {
		t.Errorf("single: %q, %v", msgs, err)
	}
	if f.model != "small" {
		t.Errorf("default model not used: %q", f.model)
	}
	msgs, _ = Generate(context.Background(), f, Request{Model: "big", Variants: true})
	if len(msgs) != 2 || f.model != "big" {
		t.Errorf("variants: %q, model %q", msgs, f.model)
	}
	f.reply = " "
	if _, err := Generate(context.Background(), f, Request{}); err == nil {
		t.Error("expected error on empty reply")
	}
}
