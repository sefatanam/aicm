---
title: "Homebrew releases push to the tap with a deploy key, not a personal access token"
author: Sefat
date: 2026-11-19
branch: main
cost: { minutes: 17, tokens: 65910 }
id: l-3c2de1a8
scope: repo
status: active
src: agent
touches:
  - ".github/workflows/release.yml"
  - ".goreleaser.yaml"
  - README.md
  - scripts/build.sh
---

## Decided
The release publishes the Homebrew cask to sefatanam/homebrew-tap over SSH using a write deploy key that only covers the tap repo. A deploy key can be set up entirely from the gh CLI, doesn't expire, and gives less access than a personal access token.

## Tried and ruled out
- A personal access token in HOMEBREW_TAP_TOKEN: it can't be created through the API, so someone has to click through the browser, and it expires and quietly breaks releases.
- Reusing the gh OAuth token as the secret: it has access to the whole account, which is far too much.

## Watch out
- The aicm repo has to stay public, or the cask's download URLs fail for everyone else.
- The key is ed25519 with no passphrase; goreleaser can't use one that has a passphrase.
- The user's git config requires annotated tags; a plain `git tag v0.1.0` fails with "no tag message?".

In the code:
- .goreleaser.yaml: homebrew_casks[].repository.git.url = git@github.com:sefatanam/homebrew-tap.git, private_key = {{ .Env.HOMEBREW_TAP_DEPLOY_KEY }}
- .github/workflows/release.yml passes secrets.HOMEBREW_TAP_DEPLOY_KEY; scripts/build.sh release refuses to run without it
- To rotate the key: ssh-keygen -t ed25519 -N "", then gh repo deploy-key add key.pub -R sefatanam/homebrew-tap --allow-write, then gh secret set HOMEBREW_TAP_DEPLOY_KEY -R sefatanam/aicm < key
- To release: git tag -a vX.Y.Z -m vX.Y.Z && git push origin vX.Y.Z
