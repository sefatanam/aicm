---
title: aicm reads git config with -z so multi-line settings survive
author: Sefat
date: 2026-11-19
branch: 1-template-without-a-ticket-clear-ticket-fallback-and-a-replaceable-style-prompt
cost: { minutes: 32, tokens: 139995 }
id: l-e79c349d
scope: repo
status: active
src: agent
touches:
  - README.md
  - config.go
  - internal/git/git.go
  - internal/git/git_test.go
  - main.go
  - main_test.go
  - internal/provider/provider.go
  - internal/provider/provider_test.go
  - internal/ticket/ticket.go
  - internal/ticket/ticket_test.go
---

## Decided
- All aicm.* settings are read with `git config -z --show-scope --show-origin`, and both the effective values and `aicm config` use that one reader. With -z, multi-line values arrive intact, and the scope and origin show where each setting comes from.

## Tried and ruled out
- Splitting plain `git config --get-regexp` output on newlines: a multi-line aicm.prompt kept only its first line, and its later lines were read as bogus keys. The first test missed it because it only checked the first line.

## Watch out
- Entries come in precedence order (system, global, local), so the last entry for a key is the one in effect; an earlier one is what it overrides.
- Tests set global config through GIT_CONFIG_GLOBAL pointing at a temp file, so `git config --global` works in tests without touching ~/.gitconfig.

In the code:
- internal/git/git.go: ConfigEntries, parseEntries, Config
- config.go: showConfig (the `aicm config` command)
- main_test.go: env.global, TestConfigCommand, TestStylePrompt
