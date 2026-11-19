package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/sefatanam/aicm/internal/git"
	"github.com/sefatanam/aicm/internal/provider"
	"github.com/sefatanam/aicm/internal/ticket"
)

// setting is one aicm.* key as `aicm config` lists it.
type setting struct {
	name string // as documented, e.g. ticketPattern
	def  func(cfg map[string]string) string
}

var settings = []setting{
	{"provider", func(map[string]string) string { return provider.Default }},
	{"model", func(cfg map[string]string) string {
		p, err := provider.Get(firstNonEmpty(cfg["provider"], provider.Default))
		if err != nil {
			return "-"
		}
		return p.DefaultModel()
	}},
	{"format", func(map[string]string) string { return ticket.DefaultTemplate + " when a ticket is found" }},
	{"ticketPattern", func(map[string]string) string { return ticket.DefaultPattern }},
	{"prompt", func(map[string]string) string { return "Conventional Commits" }},
	{"rules", func(map[string]string) string { return "-" }},
}

// showConfig prints every aicm setting in effect, where it comes from, and
// what it means for the current branch. It never calls the provider.
func showConfig(w io.Writer, o options) error {
	entries := git.ConfigEntries()
	cfg := git.Config()
	inRepo := git.EnsureRepo() == nil
	branch := ""
	if inRepo {
		branch = git.Branch()
	}

	wd, _ := os.Getwd()
	switch {
	case !inRepo:
		fmt.Fprintf(w, "aicm settings (%s is not a git repository; global only)\n\n", home(wd))
	case branch == "":
		fmt.Fprintf(w, "aicm settings for %s (detached HEAD)\n\n", home(wd))
	default:
		fmt.Fprintf(w, "aicm settings for %s (branch %s)\n\n", home(wd), branch)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, s := range settings {
		var set []git.Entry
		for _, e := range entries {
			if e.Key == strings.ToLower(s.name) {
				set = append(set, e)
			}
		}
		if len(set) == 0 {
			fmt.Fprintf(tw, "  %s\t%s\tdefault\n", s.name, s.def(cfg))
			continue
		}
		cur := set[len(set)-1]
		source := cur.Scope
		if cur.Scope != "local" {
			source += "  " + home(cur.Origin)
		}
		if len(set) > 1 {
			prev := set[len(set)-2]
			source += fmt.Sprintf("  (overrides %s: %s)", prev.Scope, oneLine(prev.Value))
		}
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", s.name, oneLine(cur.Value), source)
	}
	tw.Flush()

	if !inRepo {
		return nil
	}
	fmt.Fprintln(w, "\nOn this branch:")
	tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()

	tkt, from := o.ticket, "from -t"
	if tkt == "" {
		var err error
		if tkt, err = ticket.FromBranch(branch, cfg["ticketpattern"]); err != nil {
			fmt.Fprintf(tw, "  ticket\terror: %v\n", err)
			return nil
		}
		from = "from branch name"
	}
	if tkt == "" {
		fmt.Fprintf(tw, "  ticket\tnone\n")
	} else {
		fmt.Fprintf(tw, "  ticket\t%s (%s)\n", tkt, from)
	}

	tmpl, apply, err := ticket.Resolve(cfg["format"], tkt)
	switch {
	case err != nil:
		fmt.Fprintf(tw, "  subject\terror: %v\n", err)
	case apply:
		fmt.Fprintf(tw, "  subject\t%s\n", ticket.Apply(tmpl, tkt, "<message>"))
		fmt.Fprintf(tw, "  offers\t2 versions (with and without type)\n")
	default:
		style := "Conventional Commits"
		if strings.TrimSpace(cfg["prompt"]) != "" {
			style = "your aicm.prompt style"
		}
		fmt.Fprintf(tw, "  subject\t<message> in %s\n", style)
		fmt.Fprintf(tw, "  offers\t1 message\n")
	}
	return nil
}

// oneLine shortens a multi-line value to its first line.
func oneLine(v string) string {
	first, rest, ok := strings.Cut(v, "\n")
	if !ok {
		return v
	}
	n, unit := strings.Count(rest, "\n")+1, "lines"
	if n == 1 {
		unit = "line"
	}
	return fmt.Sprintf("%s (+%d %s)", first, n, unit)
}

// home replaces the home directory prefix of path with ~.
func home(path string) string {
	h, err := os.UserHomeDir()
	if err != nil || h == "" {
		return path
	}
	if path == h {
		return "~"
	}
	if rel, ok := strings.CutPrefix(path, h+string(filepath.Separator)); ok {
		return "~" + string(filepath.Separator) + rel
	}
	return path
}
