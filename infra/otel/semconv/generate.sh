#!/usr/bin/env bash
# Regenerate the Go package from the registry.
#
#   ./generate.sh            regenerate attribute_group.go and webitelconv/
#   ./generate.sh --check    fail if the committed output is stale (CI runs this)
#
# Downloads a pinned Weaver release; no Rust toolchain needed. The Go templates
# under ./templates are a vendored copy of the ones opentelemetry-go generates
# its own semconv packages with.
set -euo pipefail

WEAVER_VERSION="v0.25.1"

# Vendored from opentelemetry-go@58db4c898f5b5594f8ba78f156475bf48486e2f2
# (v1.46.0), semconv/templates/registry/go. Local changes: in metric.go.j2 the
# metricpool import path (upstream's is internal to its module, so
# ./internal/metricpool carries a copy) and a guard so a namespace with only
# observable metrics does not import context/metricpool unused; in weaver.yaml
# our metrics in the instrument map.
TEMPLATES="./templates"

# Every attribute we generate must carry this prefix. Putting a Webitel name in
# an upstream namespace is the mistake this registry exists to prevent.
NAMESPACE="webitel."

cd "$(dirname "$0")"

CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/webitel-weaver/${WEAVER_VERSION}"
WEAVER="${CACHE}/weaver"

# Digests recorded from the v0.25.1 release. Fetching the .sha256 from the same
# URL as the tarball only proves the download was not truncated; pinning here
# means a re-published asset fails instead of being silently trusted.
weaver_digest() {
  case "$1" in
    aarch64-apple-darwin)      echo d9e0c301077648c83c22bd17d0bfc7688ec134085ec5f673c0db69f3052a9ec5 ;;
    x86_64-apple-darwin)       echo 4185ff7b57e9de46ad602df2412d3ada12d67be35d97e2c2f52007e302b7fc90 ;;
    aarch64-unknown-linux-gnu) echo c304b535794f36ab718e73244acc27c127653e09354d5836291c8d99f8cecad5 ;;
    x86_64-unknown-linux-gnu)  echo a24f8fc17f120c3bca8ef540b907984a93875a4c4d5c9fc0604d1317be08b7bf ;;
    *) return 1 ;;
  esac
}

host_target() {
  local os arch
  os="$(uname -s)"; arch="$(uname -m)"
  case "${os}/${arch}" in
    Darwin/arm64)  echo aarch64-apple-darwin ;;
    Darwin/x86_64) echo x86_64-apple-darwin ;;
    Linux/aarch64) echo aarch64-unknown-linux-gnu ;;
    Linux/x86_64)  echo x86_64-unknown-linux-gnu ;;
    *) echo "unsupported platform ${os}/${arch}" >&2; exit 1 ;;
  esac
}

install_weaver() {
  [ -x "$WEAVER" ] && return 0

  local target want url tmp got
  target="$(host_target)"
  want="$(weaver_digest "$target")"
  url="https://github.com/open-telemetry/weaver/releases/download/${WEAVER_VERSION}/weaver-${target}.tar.xz"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN

  echo "fetching weaver ${WEAVER_VERSION} (${target})"
  curl -sSLf --retry 3 -o "${tmp}/w.tar.xz" "${url}"

  got="$(shasum -a 256 "${tmp}/w.tar.xz" | awk '{print $1}')"
  if [ "$want" != "$got" ]; then
    echo "weaver checksum mismatch: expected ${want}, got ${got}" >&2
    exit 1
  fi

  tar -xJf "${tmp}/w.tar.xz" -C "${tmp}"
  mkdir -p "${CACHE}"
  install -m 0755 "${tmp}/weaver-${target}/weaver" "${WEAVER}"
}

# Renders into $1. Never writes into the package directory, so --check cannot
# destroy a hand-edit it was asked to report.
render() {
  local out="$1"

  "$WEAVER" registry check -r ./registry --quiet
  "$WEAVER" registry generate -r ./registry -t "$TEMPLATES" --quiet go "$out"
  gofmt -w "$out"

  # The templates come from opentelemetry-go, the conventions in them do not.
  find "$out" -name '*.go' -exec sed -i.bak \
    -e 's|^// Copyright The OpenTelemetry Authors$|// Copyright (c) 2024 Webitel|' \
    -e 's|^// SPDX-License-Identifier: Apache-2.0$|// SPDX-License-Identifier: MIT|' {} +
  find "$out" -name '*.bak' -delete

  # A filter that matches nothing still exits 0, and the stale committed file
  # would then pass every check below it.
  if [ ! -s "$out/attribute_group.go" ]; then
    echo "weaver produced no attribute_group.go — the registry is not resolving" >&2
    exit 1
  fi

  # A template edit that reintroduces an upstream-internal import would only
  # fail later, at go build. Catch it here.
  local internal
  internal="$(grep -rho --include='*.go' '"go\.opentelemetry\.io/otel/[^"]*internal/[^"]*"' "$out" || true)"
  if [ -n "$internal" ]; then
    echo "generated code imports an internal upstream package:" >&2
    echo "$internal" | sort -u | sed 's/^/  /' >&2
    echo "Go forbids that from here; point the template at ./internal instead." >&2
    exit 1
  fi

  local stray
  stray="$({
      grep -rho --include='*.go' 'attribute\.Key("[^"]*"' "$out" | sed 's/.*"\(.*\)"/\1/'
      grep -rhA1 --include='*.go' 'Name() string {' "$out" | grep -o 'return "[^"]*"' | sed 's/return "//;s/"$//'
    } | grep -v "^${NAMESPACE}" || true)"
  if [ -n "$stray" ]; then
    echo "these generated names are not in the ${NAMESPACE%.} namespace:" >&2
    echo "$stray" | sed 's/^/  /' >&2
    echo "upstream conventions belong to go.opentelemetry.io/otel/semconv, not here" >&2
    exit 1
  fi
}

# Copies the generated artefacts currently in the package into $1, so --check
# can diff whole trees and notice files that appeared or disappeared.
collect_current() {
  local dst="$1" d
  [ -f attribute_group.go ] && cp attribute_group.go "$dst/"
  for d in ./*conv; do
    [ -d "$d" ] && cp -R "$d" "$dst/"
  done
  return 0
}

install_weaver

out="$(mktemp -d)"
cur="$(mktemp -d)"
trap 'rm -rf "$out" "$cur"' EXIT

case "${1:-}" in
  --check)
    render "$out"
    collect_current "$cur"
    if ! diff -ru "$cur" "$out"; then
      echo "generated output is out of date — run ./generate.sh and commit the result" >&2
      exit 1
    fi
    echo "generated output is up to date"
    ;;
  "")
    render "$out"
    rm -f attribute_group.go
    for d in ./*conv; do [ -d "$d" ] && rm -rf "$d"; done
    (cd "$out" && tar cf - .) | tar xf -
    echo "generated attribute_group.go and webitelconv/"
    ;;
  *) echo "unknown option: $1" >&2; exit 1 ;;
esac
