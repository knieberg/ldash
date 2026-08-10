#!/usr/bin/env bash
# Install ldash prebuilt binary from GitHub releases.
set -euo pipefail

VERSION="${LDASH_VERSION:-latest}"
INSTALL_DIR="${LDASH_INSTALL_DIR:-$HOME/.local/bin}"
REPO="knieberg/ldash"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

echo "ldash installer"
echo "  version: $VERSION"
echo "  target:  $INSTALL_DIR/ldash"
echo

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required. Install curl or build from source:" >&2
  echo "  git clone https://github.com/${REPO}.git && cd ldash && make install-local" >&2
  exit 1
fi

if [[ "$VERSION" == "latest" ]]; then
  API="https://api.github.com/repos/${REPO}/releases/latest"
else
  API="https://api.github.com/repos/${REPO}/releases/tags/${VERSION}"
fi

RELEASE_JSON="$(curl -fsSL "$API" 2>/dev/null || true)"
if [[ -z "$RELEASE_JSON" ]]; then
  echo "No GitHub release found for ${VERSION}." >&2
  echo "Build from source instead:" >&2
  echo "  git clone https://github.com/${REPO}.git && cd ldash && make install-local" >&2
  exit 1
fi

TAG="$(printf '%s' "$RELEASE_JSON" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": "([^"]+)".*/\1/')"
ASSET="ldash_${TAG#v}_${OS}_${ARCH}.tar.gz"
URL="$(printf '%s' "$RELEASE_JSON" | grep -o "https://[^\"]*${ASSET}[^\"]*" | head -1)"
if [[ -z "$URL" ]]; then
  echo "Release asset ${ASSET} not found for ${TAG}." >&2
  exit 1
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
curl -fsSL "$URL" -o "$TMP/archive.tar.gz"
tar -xzf "$TMP/archive.tar.gz" -C "$TMP"
install -d "$INSTALL_DIR"
install -m 755 "$TMP/ldash" "$INSTALL_DIR/ldash"

echo "Installed ldash ${TAG} to ${INSTALL_DIR}/ldash"
echo
echo "Next steps:"
echo "  ldash config init"
echo "  chmod 600 ~/.config/ldash/config.yaml"
