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

With no ticket, aicm writes a plain [Conventional Commits](https://www.conventionalcommits.org) message (`feat`, `fix`, `chore`, `refactor`, …).

## Configuration

Settings use `git config`: set them once with `--global`, or per repo without it. Repo settings take priority over global ones.

| Key | Default | Description |
|---|---|---|
| `aicm.provider` | `claude` | AI provider (currently `claude`) |
| `aicm.model` | provider default (`haiku` for Claude) | Model name (for Claude: `haiku`, `sonnet`, `opus`, or a full model ID) |
| `aicm.format` | `{ticket}: {message}` | Subject template used when there is a ticket |
| `aicm.ticketPattern` | `[A-Z][A-Z0-9]+-\d+` | Regex that finds the ticket in the branch name |
| `aicm.rules` | — | Extra style instructions for the model |

```sh
git config --global aicm.model sonnet
git config aicm.format "[{ticket}] {message}"            # [AXE-123] feat: ...
git config aicm.rules "no type prefix, max 50 chars"
```

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
