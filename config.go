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

// Setting keys, as git lowercases them after "aicm.".
const (
	keyProvider      = "provider"
	keyModel         = "model"
	keyFormat        = "format"
	keyTicketPattern = "ticketpattern"
	keyPrompt        = "prompt"
	keyRules         = "rules"
)

// settings lists the keys `aicm config` shows, in display order, with the
// name the README documents for each.
var settings = []struct{ key, name string }{
	{keyProvider, "provider"},
	{keyModel, "model"},
	{keyFormat, "format"},
	{keyTicketPattern, "ticketPattern"},
	{keyPrompt, "prompt"},
	{keyRules, "rules"},
}

// defaultFor describes the value used when a setting is unset.
func defaultFor(key string, cfg map[string]string) string {
	switch key {
	case keyProvider:
		return provider.Default
	case keyModel:
		p, err := providerFor("", cfg)
		if err != nil {
			return "-"
		}
		return p.DefaultModel()
	case keyFormat:
		return ticket.DefaultTemplate + " when a ticket is found"
	case keyTicketPattern:
		return ticket.DefaultPattern
	case keyPrompt:
		return provider.DefaultStyleName
	}
	return "-"
}

// showConfig prints the aicm settings in effect, where each comes from, and
// what they mean on the current branch. It never calls the provider.
func showConfig(w io.Writer, o options) error {
	inRepo := git.EnsureRepo() == nil
	entries, err := git.ConfigEntries()
	if err != nil {
		return err
	}
	if !inRepo {
		entries = withoutScope(entries, "system")
	}
	cfg := git.Values(entries)

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

	type block struct{ name, value string }
	var multiline []block
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	known := map[string]bool{}
	for _, s := range settings {
		known[s.key] = true
		set := git.Lookup(entries, s.key)
		if len(set) == 0 {
			fmt.Fprintf(tw, "  %s\t%s\tdefault\n", s.name, defaultFor(s.key, cfg))
			continue
		}
		value := oneLine(set[len(set)-1].Value)
		if strings.Contains(set[len(set)-1].Value, "\n") {
			value = fmt.Sprintf("(%d lines, shown below)", strings.Count(set[len(set)-1].Value, "\n")+1)
			multiline = append(multiline, block{s.name, set[len(set)-1].Value})
		}
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", s.name, value, sourceOf(set))
	}
	tw.Flush()

	for _, b := range multiline {
		fmt.Fprintf(w, "\n%s:\n", b.name)
		for _, line := range strings.Split(b.value, "\n") {
			fmt.Fprintf(w, "    %s\n", line)
		}
	}

	var unknown []git.Entry
	for _, e := range entries {
		if !known[e.Key] {
			unknown = append(unknown, e)
		}
	}
	if len(unknown) > 0 {
		fmt.Fprintln(w, "\nNot settings aicm uses (check the spelling):")
		for _, e := range unknown {
			fmt.Fprintf(w, "  aicm.%s = %s  (%s)\n", e.Key, oneLine(e.Value), e.Scope)
		}
	}

	if !inRepo {
		return nil
	}
	tkt, source, err := findTicket(o.ticket, branch, cfg[keyTicketPattern])
	if err != nil {
		return err
	}
	tmpl, applies, err := ticket.Resolve(cfg[keyFormat], tkt)
	if err != nil {
		return err
	}

	fmt.Fprintln(w, "\nOn this branch:")
	bw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if tkt == "" {
		fmt.Fprintln(bw, "  ticket\tnone")
	} else {
		fmt.Fprintf(bw, "  ticket\t%s (%s)\n", tkt, source)
	}
	if applies {
		fmt.Fprintf(bw, "  subject\t%s\n", ticket.Apply(tmpl, tkt, "<message>"))
		fmt.Fprintln(bw, "  offers\t2 versions (in your style, and a plain subject)")
	} else {
		style := provider.DefaultStyleName
		if cfg[keyPrompt] != "" {
			style = "your aicm.prompt style"
		}
		fmt.Fprintf(bw, "  subject\t<message> in %s\n", style)
		fmt.Fprintln(bw, "  offers\t1 message")
	}
	return bw.Flush()
}

// sourceOf describes where the effective setting comes from, and the value it
// overrides when more than one scope sets it.
func sourceOf(set []git.Entry) string {
	cur := set[len(set)-1]
	src := cur.Scope
	if cur.Scope != "local" {
		src += "  " + home(cur.Origin)
	}
	if len(set) > 1 {
		prev := set[len(set)-2]
		src += fmt.Sprintf("  (overrides %s: %s)", prev.Scope, oneLine(prev.Value))
	}
	return src
}

// withoutScope drops the entries that came from scope.
func withoutScope(entries []git.Entry, scope string) []git.Entry {
	var kept []git.Entry
	for _, e := range entries {
		if e.Scope != scope {
			kept = append(kept, e)
		}
	}
	return kept
}

// oneLine shortens a multi-line value to its first line, for table cells.
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
