#!/usr/bin/env bash
# Build, test and release aicm. Used by CI and locally.
#
#   ./scripts/build.sh            check + build (default)
#   ./scripts/build.sh check      gofmt, go vet, go test
#   ./scripts/build.sh build      build ./bin/aicm for this machine
#   ./scripts/build.sh install    install aicm into $(go env GOPATH)/bin
#   ./scripts/build.sh snapshot   build release archives for every platform into ./dist (no publish)
#   ./scripts/build.sh release    publish a release for the current tag (CI only; needs GITHUB_TOKEN, HOMEBREW_TAP_TOKEN)
#   ./scripts/build.sh clean      remove ./bin and ./dist
set -euo pipefail

cd "$(dirname "$0")/.."

GORELEASER_VERSION="v2.20.2"
VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
LDFLAGS="-s -w -X main.version=${VERSION}"

log() { printf '\033[1;34m==>\033[0m %s\n' "$*"; }
die() { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

run_goreleaser() {
  if type -P goreleaser >/dev/null; then
    goreleaser "$@"
  else
    go run "github.com/goreleaser/goreleaser/v2@${GORELEASER_VERSION}" "$@"
  fi
}

check() {
  log "gofmt"
  local unformatted
  unformatted="$(gofmt -l .)"
  [[ -z "$unformatted" ]] || die "files need gofmt:"$'\n'"$unformatted"
  log "go vet"
  go vet ./...
  log "go test"
  go test ./...
}

build() {
  local out="bin/aicm"
  [[ "$(go env GOOS)" == "windows" ]] && out="bin/aicm.exe"
  log "build ${out} (${VERSION})"
  CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o "$out" .
}

install() {
  log "install aicm (${VERSION}) to $(go env GOPATH)/bin"
  CGO_ENABLED=0 go install -trimpath -ldflags "$LDFLAGS" .
}

snapshot() {
  log "goreleaser snapshot"
  run_goreleaser release --snapshot --clean
}

release() {
  git describe --exact-match --tags >/dev/null 2>&1 || die "HEAD is not tagged; push a v* tag to release"
  [[ -n "${GITHUB_TOKEN:-}" ]] || die "GITHUB_TOKEN is not set"
  [[ -n "${HOMEBREW_TAP_TOKEN:-}" ]] || die "HOMEBREW_TAP_TOKEN is not set"
  check
  log "goreleaser release"
  run_goreleaser release --clean
}

clean() {
  log "clean"
  rm -rf bin dist
}

case "${1:-all}" in
  all)      check; build ;;
  check)    check ;;
  build)    build ;;
  install)  install ;;
  snapshot) snapshot ;;
  release)  release ;;
  clean)    clean ;;
  -h|--help|help) sed -n '2,10p' "$0" | sed 's/^# \{0,1\}//' ;;
  *) die "unknown command: $1 (try --help)" ;;
esac
