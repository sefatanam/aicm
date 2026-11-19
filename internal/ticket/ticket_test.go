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

func TestApply(t *testing.T) {
	cases := []struct{ tmpl, ticket, msg, want string }{
		{"", "AXE-1", "feat: add login", "AXE-1: feat: add login"},
		{"[{ticket}] {message}", "AXE-1", "add login", "[AXE-1] add login"},
		{"", "AXE-1", "fix: bug\n\nlonger body", "AXE-1: fix: bug\n\nlonger body"},
	}
	for _, c := range cases {
		if got := Apply(c.tmpl, c.ticket, c.msg); got != c.want {
			t.Errorf("Apply(%q, %q, %q) = %q, want %q", c.tmpl, c.ticket, c.msg, got, c.want)
		}
	}
}
