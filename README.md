# aicm

AI commit messages from your staged changes. By default it uses your **Claude subscription**, so no API key is needed.

```console
$ git add .
$ aicm
    feat(auth): add login with session cookie

Commit? [y]es / [e]dit / [r]egenerate / [n]o: y
```

The default provider is Claude. aicm calls the [Claude Code](https://claude.com/claude-code) CLI under the hood, so it uses the login you already have. Haiku is the default model, so a message usually takes 2–4 seconds. Support for Ollama, Codex and Gemini is planned.

## Requirements

- git
- For the Claude provider: Claude Code installed and logged in (run `claude` once)

## Install

```sh
# Homebrew (macOS, Linux)
brew install --cask sefatanam/tap/aicm

# Go
go install github.com/sefatanam/aicm@latest
```

Or download a binary for macOS, Linux or Windows from [Releases](https://github.com/sefatanam/aicm/releases).

## Usage

```sh
aicm             # write a message, then commit / edit / regenerate / quit
aicm -y          # commit without asking
aicm -p          # print the message only
aicm -a          # stage tracked changes first (like git commit -a)
aicm -t AXE-123  # use this ticket ID
aicm -m sonnet   # use another model for this run
aicm -P claude   # use another provider for this run
```

When stdout is piped, aicm only prints the message:

```sh
git commit -m "$(aicm)"
```

## Tickets

If your branch name contains a ticket ID, aicm puts it in the subject and offers two variants to choose from:

```console
$ git switch -c feature/AXE-123-login
$ aicm
[1] AXE-123: feat(auth): add login with session cookie
[2] AXE-123: Add login with session cookie

Pick a message [1-2] (default 1):
```

How the subject is shaped:

| Ticket found? | `aicm.format` | Result |
|---|---|---|
| yes | unset | `AXE-123: feat(auth): …` (default `{ticket}: {message}`) |
| yes | `[{ticket}] {message}` | `[AXE-123] feat(auth): …` |
| no | unset, or mentions `{ticket}` | plain [Conventional Commits](https://www.conventionalcommits.org): `feat(auth): …` |
| either | `[web] {message}` (no `{ticket}`) | `[web] feat(auth): …` |

Whenever a format is applied, aicm offers two versions to pick from: with and without the Conventional Commits type.

## Configuration

Settings use `git config`: set them once with `--global`, or per repo without it. Repo settings take priority over global ones.

Check what's in effect with `aicm config`. It lists each setting, its value and where it comes from (this repo, global or default), then what that means on the current branch:

```console
$ aicm config
aicm settings for ~/work/shop (branch feature/AXE-123-login)

  provider       claude                default
  model          sonnet                local  (overrides global: haiku)
  format         [{ticket}] {message}  global  ~/.gitconfig
  ticketPattern  [A-Z][A-Z0-9]+-\d+    default
  prompt         Conventional Commits  default
  rules          lowercase, no period  global  ~/.gitconfig

On this branch:
  ticket   AXE-123 (from branch name)
  subject  [AXE-123] <message>
  offers   2 versions (with and without type)
```

| Key | Default | Description |
|---|---|---|
| `aicm.provider` | `claude` | AI provider (currently `claude`) |
| `aicm.model` | provider default (`haiku` for Claude) | Model name (for Claude: `haiku`, `sonnet`, `opus`, or a full model ID) |
| `aicm.format` | `{ticket}: {message}` when there is a ticket | Subject template. Must contain `{message}`; one that contains `{ticket}` is only used when a ticket is found |
| `aicm.ticketPattern` | `[A-Z][A-Z0-9]+-\d+` | Regex that finds the ticket in the branch name |
| `aicm.prompt` | Conventional Commits | Replaces the built-in style instructions. aicm still keeps the output to just the message and adds tickets itself |
| `aicm.rules` | — | Extra style instructions, applied on top of the style and taking priority |

```sh
git config --global aicm.model sonnet
git config aicm.format "[{ticket}] {message}"            # [AXE-123] feat: ...
git config aicm.format "[web] {message}"                 # every commit: [web] feat: ...
git config aicm.rules "no type prefix, max 50 chars"
git config --global aicm.prompt $'Use gitmoji.\nOne line, at most 50 characters.'
```

Longer prompts are easiest to write with `git config --global --edit`.

### Team setup

Repo-level `git config` isn't committed, so share your convention as a block each teammate runs once:

```sh
git config aicm.format "{ticket}: {message}"
git config aicm.rules "lowercase subject, max 72 chars"
```

Or commit a file with an `[aicm]` section (e.g. `.aicmconfig`) and have each teammate include it once: `git config include.path ../.aicmconfig`.

## Development

Requires Go. CI and local builds both use `scripts/build.sh`:

```sh
./scripts/build.sh            # check + build (default)
./scripts/build.sh check      # gofmt, go vet, go test
./scripts/build.sh build      # ./bin/aicm for this machine
./scripts/build.sh install    # install into $(go env GOPATH)/bin
./scripts/build.sh snapshot   # release archives for all platforms in ./dist, no publish
./scripts/build.sh clean      # remove ./bin and ./dist
```

GoReleaser is used if it is installed. If not, the script runs a pinned version through `go run`.

## Releasing

Push a `v*` tag (`git tag v0.1.0 && git push --tags`). The release workflow runs `./scripts/build.sh release`, which tests and then runs goreleaser, which publishes binaries to GitHub Releases and updates the cask in `sefatanam/homebrew-tap`. It needs a `HOMEBREW_TAP_DEPLOY_KEY` repository secret: the private half of an SSH deploy key that has write access to `sefatanam/homebrew-tap`.
