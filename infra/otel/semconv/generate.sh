#!/usr/bin/env bash
# Regenerate the Go semantic-convention package from the registry.
#
#   ./generate.sh                             regenerate attribute.go
#   ./generate.sh --check                     fail if attribute.go is stale
#   ./generate.sh --set-namespace com.webitel change our prefix, then regenerate
#
# Downloads a pinned Weaver release; no Rust toolchain needed.
set -euo pipefail

WEAVER_VERSION="v0.25.1"
cd "$(dirname "$0")"

CACHE="${XDG_CACHE_HOME:-$HOME/.cache}/webitel-weaver/${WEAVER_VERSION}"
WEAVER="${CACHE}/weaver"

current_namespace() {
  sed -n 's/^[[:space:]]*namespace:[[:space:]]*\([A-Za-z0-9._-]*\).*/\1/p' \
    templates/go/weaver.yaml | head -1
}

# Schema URLs come from the manifest, so the pin has one source.
manifest_schema_url() {
  sed -n 's|^schema_url:[[:space:]]*\(.*\)|\1|p' registry/manifest.yaml | head -1
}
upstream_schema_url() {
  sed -n 's|^[[:space:]]*-[[:space:]]*schema_url:[[:space:]]*\(.*\)|\1|p' \
    registry/manifest.yaml | head -1
}

install_weaver() {
  [ -x "$WEAVER" ] && return 0

  local os arch target
  os="$(uname -s)"; arch="$(uname -m)"
  case "${os}/${arch}" in
    Darwin/arm64)  target="aarch64-apple-darwin" ;;
    Darwin/x86_64) target="x86_64-apple-darwin" ;;
    Linux/aarch64) target="aarch64-unknown-linux-gnu" ;;
    Linux/x86_64)  target="x86_64-unknown-linux-gnu" ;;
    *) echo "unsupported platform ${os}/${arch}" >&2; exit 1 ;;
  esac

  local url tmp
  url="https://github.com/open-telemetry/weaver/releases/download/${WEAVER_VERSION}/weaver-${target}.tar.xz"
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' RETURN

  echo "fetching weaver ${WEAVER_VERSION} (${target})"
  curl -sSLf --retry 3 -o "${tmp}/w.tar.xz" "${url}"
  curl -sSLf --retry 3 -o "${tmp}/w.sha256" "${url}.sha256"

  # Published format is "<digest> *<filename>"; compare digests only.
  local want got
  want="$(awk '{print $1}' "${tmp}/w.sha256")"
  got="$(shasum -a 256 "${tmp}/w.tar.xz" | awk '{print $1}')"
  if [ "$want" != "$got" ]; then
    echo "weaver checksum mismatch: expected ${want}, got ${got}" >&2
    exit 1
  fi

  tar -xJf "${tmp}/w.tar.xz" -C "${tmp}"
  mkdir -p "${CACHE}"
  install -m 0755 "${tmp}/weaver-${target}/weaver" "${WEAVER}"
}

set_namespace() {
  local new="$1" old
  old="$(current_namespace)"
  [ -n "$old" ] || { echo "could not read current namespace" >&2; exit 1; }
  if [ "$old" = "$new" ]; then echo "namespace already ${new}"; return 0; fi

  # Keep the prefix safe both as a sed replacement and as an attribute segment.
  case "$new" in
    *[!a-z0-9_.]*|.*|*.) echo "invalid namespace: ${new}" >&2; exit 1 ;;
  esac

  # Refuse a namespace that collides with one upstream owns. Putting our names
  # in an upstream namespace is the exact mistake this registry exists to stop,
  # and it also makes the rewrite ambiguous: with namespace "db", upstream's
  # own db.collection.name would be treated as ours.
  case ".${new}." in
    .db.*|.rpc.*|.http.*|.net.*|.network.*|.client.*|.server.*|.service.*|.messaging.*|.url.*|.error.*|.user_agent.*|.otel.*|.telemetry.*)
      echo "refusing namespace '${new}': that is an upstream OpenTelemetry namespace" >&2
      exit 1 ;;
  esac

  echo "namespace: ${old} -> ${new}"

  # No \b here: it is a GNU extension, and BSD/macOS sed silently matches
  # nothing — which made this command a no-op that still exited 0.
  #
  # manifest.yaml is excluded deliberately: its schema_url is
  # https://webitel.com/..., and a blind rewrite would corrupt the hostname.
  find registry -name '*.yaml' ! -name 'manifest.yaml' -exec \
    sed -i.bak "s/${old}\./${new}./g" {} +

  # Our own metric names in the boundaries map, then the setting itself.
  sed -i.bak "s/${old}\./${new}./g" templates/go/weaver.yaml
  sed -i.bak "s/^\([[:space:]]*\)namespace: ${old}\$/\1namespace: ${new}/" templates/go/weaver.yaml

  find . -name '*.bak' -delete

  echo "note: attribute_test.go pins the old wire names — update it to match." >&2
}

generate() {
  # --future is deliberately omitted: it applies the newest rules to the
  # dependency too, and upstream v1.30.0 fails them. Revisit when the pin rises.
  "$WEAVER" registry check -r ./registry --quiet
  "$WEAVER" registry generate \
    -r ./registry \
    -t ./templates \
    -D "schema_url=$(manifest_schema_url)" \
    -D "upstream_schema_url=$(upstream_schema_url)" \
    --quiet \
    go .
  gofmt -w attribute.go

  # Fail loud on an empty package. Both bugs found in review produced a
  # valid-looking file with no keys in it, and exit code 0.
  local keys
  keys="$(grep -c 'attribute\.Key(' attribute.go || true)"
  if [ "${keys:-0}" -lt 10 ]; then
    echo "attribute.go has only ${keys:-0} keys — the registry filter is not matching" >&2
    exit 1
  fi
}

install_weaver

case "${1:-}" in
  --set-namespace)
    [ -n "${2:-}" ] || { echo "usage: $0 --set-namespace <prefix>" >&2; exit 1; }
    set_namespace "$2"
    generate
    ;;
  --check)
    before="$(cat attribute.go 2>/dev/null || true)"
    generate
    if [ "$before" != "$(cat attribute.go)" ]; then
      echo "attribute.go is out of date — run ./generate.sh and commit the result" >&2
      exit 1
    fi
    echo "attribute.go is up to date"
    ;;
  "") generate ;;
  *) echo "unknown option: $1" >&2; exit 1 ;;
esac
