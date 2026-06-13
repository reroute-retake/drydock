#!/bin/sh
# drydock installer — downloads the latest `dock` release into a user-owned dir.
#
#   curl -fsSL https://raw.githubusercontent.com/reroute-retake/drydock/main/install.sh | sh
#
# Env overrides:
#   DRYDOCK_BIN=/custom/bin   install location (default: ~/.local/bin)
#   VERSION=v0.3.0            pin a release (default: latest)
set -eu

OWNER=reroute-retake
REPO=drydock
BIN_DIR="${DRYDOCK_BIN:-$HOME/.local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
[ "$os" = "linux" ] || { echo "drydock currently supports Linux only (got: $os)"; exit 1; }

arch="$(uname -m)"
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "unsupported architecture: $arch"; exit 1 ;;
esac

asset="dock_${os}_${arch}.tar.gz"
if [ -n "${VERSION:-}" ]; then
  base="https://github.com/$OWNER/$REPO/releases/download/$VERSION"
else
  base="https://github.com/$OWNER/$REPO/releases/latest/download"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "downloading $asset ..."
curl -fsSL "$base/$asset" -o "$tmp/$asset"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"

echo "verifying checksum ..."
( cd "$tmp" && grep " ${asset}\$" checksums.txt | sha256sum -c - >/dev/null )

tar -xzf "$tmp/$asset" -C "$tmp" dock
mkdir -p "$BIN_DIR"
install -m 0755 "$tmp/dock" "$BIN_DIR/dock"
echo "installed dock -> $BIN_DIR/dock"

case ":$PATH:" in
  *":$BIN_DIR:"*) : ;;
  *) echo "note: $BIN_DIR is not on your PATH — add:  export PATH=\"$BIN_DIR:\$PATH\"" ;;
esac

"$BIN_DIR/dock" version || true
