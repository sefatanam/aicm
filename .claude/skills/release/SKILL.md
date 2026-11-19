---
name: release
description: Prepare and publish the next aicm version (GitHub Release + Homebrew cask). Optional arg: a version (v0.2.0) or bump (patch|minor|major).
disable-model-invocation: true
---

Cut an aicm release. Pushing a `v*` tag is the release: `.github/workflows/release.yml` runs `./scripts/build.sh release`, goreleaser publishes the six archives to GitHub Releases and commits `Casks/aicm.rb` to `sefatanam/homebrew-tap` over SSH with the `HOMEBREW_TAP_DEPLOY_KEY` deploy key. The version is stamped from the tag at build time; there is no version file to edit.

## 1. Preflight

Done when every check below passes. Stop and report the first failure; fix only with the user's go-ahead.

- On `main`, no uncommitted changes to tracked files, and `git fetch` shows `main` equal to `origin/main`.
- `./scripts/build.sh check` passes.
- `go run github.com/goreleaser/goreleaser/v2@v2.20.2 check` validates the config (use `goreleaser check` if it's installed).
- `trail check` reports no conflicts.
- `gh secret list -R sefatanam/aicm` lists `HOMEBREW_TAP_DEPLOY_KEY`, and `sefatanam/aicm` and `sefatanam/homebrew-tap` are both public (`gh repo view <repo> --json visibility`). The cask's download URLs fail for everyone else if aicm goes private.

## 2. Pick the version

Done when the user has confirmed one exact `vX.Y.Z`.

- Last release: `git describe --tags --abbrev=0`. Changes: `git log --oneline <last>..HEAD`. Nothing since the last tag → stop, there is nothing to release.
- If the argument names a version or a bump, use it. Otherwise propose one from the Conventional Commit types: `feat` → minor, `fix`/`perf`/anything else → patch, `!` or `BREAKING CHANGE` → major (before 1.0 a breaking change bumps minor).
- Show the user the proposed version and the commit list, and wait for their confirmation.

## 3. Tag and push

Use an annotated tag: this user's git config rejects lightweight tags ("no tag message?").

```sh
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
```

## 4. Watch the workflow

Done when the run has finished with success.

```sh
id=$(gh run list -R sefatanam/aicm --workflow release.yml -L1 --json databaseId -q '.[0].databaseId')
gh run watch "$id" -R sefatanam/aicm --exit-status
```

On failure, show `gh run view "$id" -R sefatanam/aicm --log-failed` and diagnose. A version that has been pushed is public: ship the fix as the next patch version. Deleting and re-pushing the same tag (`git push --delete origin vX.Y.Z`, `gh release delete vX.Y.Z`) needs the user's explicit approval.

## 5. Verify

Done when all of these hold:

- `gh release view vX.Y.Z -R sefatanam/aicm` lists the six `aicm_<os>_<arch>` archives plus `checksums.txt`.
- `gh api repos/sefatanam/homebrew-tap/contents/Casks/aicm.rb -q .content | base64 -d | grep version` shows `X.Y.Z`.
- `brew update && brew upgrade --cask sefatanam/tap/aicm` (or `brew install --cask` if it isn't installed), then `aicm --version` prints `aicm X.Y.Z`.

Report the release URL, the version, and how each check came out.
