// Command aicm writes git commit messages with an AI provider. The default
// provider, Claude, uses your Claude Code subscription instead of an API key.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/sefatanam/aicm/internal/git"
	"github.com/sefatanam/aicm/internal/provider"
	"github.com/sefatanam/aicm/internal/ticket"
	"github.com/sefatanam/aicm/internal/ui"
)

var version = "dev" // set by goreleaser via -ldflags

const usage = `aicm - AI commit messages from your staged changes

Usage:
  aicm [flags]

Flags:
  -p           print the message only (automatic when stdout is piped)
  -y           commit without asking
  -a           stage tracked changes first (git add -u)
  -t TICKET    ticket ID to prefix (default: taken from the branch name)
  -P PROVIDER  AI provider (default: aicm.provider or claude)
  -m MODEL     model (default: aicm.model or the provider's default)
  -v           print version

Config (git config [--global] <key> <value>):
  aicm.provider       AI provider (default claude)
  aicm.model          model to use (default depends on the provider; claude: haiku)
  aicm.format         subject template; with {ticket} only used when a ticket is found
                      (default "{ticket}: {message}" when a ticket is found)
  aicm.ticketPattern  regex for the ticket in the branch name (default "[A-Z][A-Z0-9]+-\d+")
  aicm.prompt         replace the built-in style instructions (Conventional Commits)
  aicm.rules          extra style instructions for the model
`

type options struct {
	print, yes, all, version bool
	ticket, provider, model  string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "aicm:", err)
		os.Exit(1)
	}
}

func run() error {
	var o options
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.BoolVar(&o.print, "p", false, "")
	flag.BoolVar(&o.yes, "y", false, "")
	flag.BoolVar(&o.all, "a", false, "")
	flag.BoolVar(&o.version, "v", false, "")
	flag.BoolVar(&o.version, "version", false, "")
	flag.StringVar(&o.ticket, "t", "", "")
	flag.StringVar(&o.provider, "P", "", "")
	flag.StringVar(&o.model, "m", "", "")
	flag.Parse()

	if o.version {
		fmt.Println("aicm", version)
		return nil
	}
	interactive := !o.print && !o.yes && ui.IsTerminal(os.Stdout) && ui.IsTerminal(os.Stdin)
	if !o.yes && !interactive {
		o.print = true
	}

	if err := git.EnsureRepo(); err != nil {
		return err
	}
	if o.all {
		if err := git.StageTracked(); err != nil {
			return err
		}
	}
	diff, stat, err := git.Staged()
	if err != nil {
		return err
	}
	if strings.TrimSpace(diff) == "" {
		return errors.New("nothing staged; stage files with `git add` or run `aicm -a`")
	}

	cfg := git.Config()
	p, err := provider.Get(firstNonEmpty(o.provider, cfg["provider"], provider.Default))
	if err != nil {
		return err
	}
	tkt := o.ticket
	if tkt == "" {
		if tkt, err = ticket.FromBranch(git.Branch(), cfg["ticketpattern"]); err != nil {
			return err
		}
	}
	tmpl, apply, err := ticket.Resolve(cfg["format"], tkt)
	if err != nil {
		return err
	}
	if o.ticket != "" && apply && !strings.Contains(tmpl, "{ticket}") {
		fmt.Fprintf(os.Stderr, "aicm: warning: aicm.format %q has no {ticket}; -t %s is not used\n", tmpl, o.ticket)
	}
	req := provider.Request{
		Model:    firstNonEmpty(o.model, cfg["model"], p.DefaultModel()),
		Diff:     diff,
		Stat:     stat,
		Style:    cfg["prompt"],
		Rules:    cfg["rules"],
		Variants: apply,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	for {
		variants, err := generate(ctx, p, req)
		if err != nil {
			return err
		}
		if apply {
			for i, v := range variants {
				variants[i] = ticket.Apply(tmpl, tkt, v)
			}
		}

		switch {
		case o.print:
			fmt.Println(variants[0])
			return nil
		case o.yes:
			return git.Commit(variants[0])
		}

		p := ui.NewPrompter(os.Stdin, os.Stderr)
		msg := variants[0]
		if len(variants) > 1 {
			msg = variants[p.Pick(variants)]
		}
		showMsg := len(variants) == 1
		for {
			switch p.Confirm(msg, showMsg) {
			case ui.Commit:
				return git.Commit(msg)
			case ui.Edit:
				edited, err := ui.EditInEditor(msg)
				if err != nil {
					return err
				}
				if edited == "" {
					return errors.New("empty message; aborted")
				}
				msg, showMsg = edited, true
				continue
			case ui.Regenerate:
			case ui.Quit:
				fmt.Fprintln(os.Stderr, "aborted")
				return nil
			}
			break // regenerate
		}
	}
}

func generate(ctx context.Context, p provider.Provider, req provider.Request) ([]string, error) {
	if !ui.IsTerminal(os.Stderr) {
		return provider.Generate(ctx, p, req)
	}
	spinCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		ui.Spin(spinCtx, os.Stderr, "writing commit message with "+p.Name()+"/"+req.Model+"…")
		close(done)
	}()
	msgs, err := provider.Generate(ctx, p, req)
	cancel()
	<-done
	return msgs, err
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
