#!/usr/bin/env bash
# Install ldash prebuilt binary from GitHub releases (placeholder — build from source until first release).
set -euo pipefail

VERSION="${LDASH_VERSION:-latest}"
INSTALL_DIR="${LDASH_INSTALL_DIR:-/usr/local/bin}"
REPO="knieberg/ldash"

echo "ldash installer"
echo "  version: $VERSION"
echo "  target:  $INSTALL_DIR/ldash"
echo
echo "No published release yet. Build from source:"
echo "  git clone https://github.com/${REPO}.git"
echo "  cd ldash && make build && make install-local"
echo
echo "Then initialize config:"
echo "  ldash config init"
exit 0
