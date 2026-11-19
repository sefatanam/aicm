package ticket

import "testing"

func TestFromBranch(t *testing.T) {
	cases := []struct{ branch, pattern, want string }{
		{"feature/AXE-123-login", "", "AXE-123"},
		{"ONE-0001", "", "ONE-0001"},
		{"bugfix/PROJ2-9_fix", "", "PROJ2-9"},
		{"main", "", ""},
		{"feature/axe-123", "", ""},
		{"gh-42-thing", `gh-\d+`, "gh-42"},
	}
	for _, c := range cases {
		got, err := FromBranch(c.branch, c.pattern)
		if err != nil {
			t.Fatalf("%q: %v", c.branch, err)
		}
		if got != c.want {
			t.Errorf("FromBranch(%q, %q) = %q, want %q", c.branch, c.pattern, got, c.want)
		}
	}
}

func TestFromBranchBadPattern(t *testing.T) {
	if _, err := FromBranch("x", "("); err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestResolve(t *testing.T) {
	cases := []struct {
		name, tmpl, ticket string
		want               string
		apply              bool
	}{
		{"no template, no ticket", "", "", "", false},
		{"no template, ticket", "", "AXE-1", DefaultTemplate, true},
		{"ticket template, no ticket", "{ticket} | {message}", "", "", false},
		{"ticket template, ticket", "{ticket} | {message}", "AXE-1", "{ticket} | {message}", true},
		{"plain template, no ticket", "[web] {message}", "", "[web] {message}", true},
		{"plain template, ticket", "[web] {message}", "AXE-1", "[web] {message}", true},
	}
	for _, c := range cases {
		got, apply, err := Resolve(c.tmpl, c.ticket)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want || apply != c.apply {
			t.Errorf("%s: Resolve(%q, %q) = %q, %v; want %q, %v", c.name, c.tmpl, c.ticket, got, apply, c.want, c.apply)
		}
	}
}

func TestResolveNeedsMessage(t *testing.T) {
	if _, _, err := Resolve("[{ticket}]", "AXE-1"); err == nil {
		t.Fatal("expected error for template without {message}")
	}
}

func TestApply(t *testing.T) {
	cases := []struct{ tmpl, ticket, msg, want string }{
		{"", "AXE-1", "feat: add login", "AXE-1: feat: add login"},
		{"[{ticket}] {message}", "AXE-1", "add login", "[AXE-1] add login"},
		{"", "AXE-1", "fix: bug\n\nlonger body", "AXE-1: fix: bug\n\nlonger body"},
		{"[web] {message}", "", "feat: add login", "[web] feat: add login"},
	}
	for _, c := range cases {
		if got := Apply(c.tmpl, c.ticket, c.msg); got != c.want {
			t.Errorf("Apply(%q, %q, %q) = %q, want %q", c.tmpl, c.ticket, c.msg, got, c.want)
		}
	}
}
