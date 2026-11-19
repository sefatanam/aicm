# 1. Call the Claude Code CLI instead of the Anthropic API

Date: 2026-11-19

## Status

Accepted

## Context

aicm's default provider, Claude, must work with a Claude subscription (Pro/Max) and no API key. The Anthropic API needs an API key and bills per token separately from the subscription. The Claude Code CLI (`claude -p`) runs non-interactively and authenticates with the user's existing subscription login.

## Decision

The Claude provider runs `claude -p` as a subprocess and passes the staged diff on stdin:

```
claude -p --model <m> --tools "" --no-session-persistence \
       --setting-sources "" --strict-mcp-config --system-prompt <prompt>
```

- `--tools ""`, `--setting-sources ""` and `--strict-mcp-config` turn off tools, user hooks and MCP servers, so the call is fast and has no side effects.
- `--bare` is **not** used: it only accepts `ANTHROPIC_API_KEY` and would break subscription login.

## Consequences

- No API key and no extra billing: usage counts against the user's subscription.
- Claude Code must be installed and logged in, and the provider will break if the CLI flags change. Flags are pinned in `internal/provider/claude.go` and covered by `TestClaudeArgs`.
- Each run pays the CLI's start-up cost (~0.5 s) on top of model latency. That is acceptable next to the 2–5 s the model itself takes.
