---
title: "Homebrew releases push to the tap with a deploy key, not a personal access token"
author: Sefat
date: 2026-11-19
branch: main
cost: { minutes: 25, tokens: 84826 }
id: l-3c2de1a8
scope: repo
status: active
src: agent
touches:
  - ".github/workflows/release.yml"
  - ".goreleaser.yaml"
  - README.md
  - scripts/build.sh
  - ".gitattributes"
  - ".github/workflows/test.yml"
---

## Decided
The release publishes the Homebrew cask to sefatanam/homebrew-tap over SSH using a write deploy key that only covers the tap repo. A deploy key can be set up entirely from the gh CLI, doesn't expire, and gives less access than a personal access token.
- The repo forces LF line endings for every file. Windows runners check out with core.autocrlf=true, so files arrive with CRLF and gofmt -l flags every .go file, failing `build.sh check` on Windows only.
- The test matrix has fail-fast off. With it on, the Windows failure cancelled the Linux and macOS jobs, which made it look as if every OS was broken.

## Tried and ruled out
- A personal access token in HOMEBREW_TAP_TOKEN: it can't be created through the API, so someone has to click through the browser, and it expires and quietly breaks releases.
- Reusing the gh OAuth token as the secret: it has access to the whole account, which is far too much.
- Reproducing with act: act only runs Linux containers, so it can't run the windows-latest job. It's still useful to show that the Linux job passes and the code itself is fine.

## Watch out
- The aicm repo has to stay public, or the cask's download URLs fail for everyone else.
- The key is ed25519 with no passphrase; goreleaser can't use one that has a passphrase.
- The user's git config requires annotated tags; a plain `git tag v0.1.0` fails with "no tag message?".

In the code:
- .goreleaser.yaml: homebrew_casks[].repository.git.url = git@github.com:sefatanam/homebrew-tap.git, private_key = {{ .Env.HOMEBREW_TAP_DEPLOY_KEY }}
- .github/workflows/release.yml passes secrets.HOMEBREW_TAP_DEPLOY_KEY; scripts/build.sh release refuses to run without it
- To rotate the key: ssh-keygen -t ed25519 -N "", then gh repo deploy-key add key.pub -R sefatanam/homebrew-tap --allow-write, then gh secret set HOMEBREW_TAP_DEPLOY_KEY -R sefatanam/aicm < key
- To release: git tag -a vX.Y.Z -m vX.Y.Z && git push origin vX.Y.Z
- Under act, the job always ends with "Job failed" / exit code 127 from setup-go's post step ("node" not found, act issue #107). That's act's cleanup, not this repo; check the ✅ on the real steps.
- To test Windows line endings on a Mac: clone with `git clone --config core.autocrlf=true` and run gofmt -l on the clone.

- .gitattributes: `* text=auto eol=lf`
- .github/workflows/test.yml: strategy.fail-fast: false
- act: `act push -j test --matrix os:ubuntu-latest`
