#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="${LDASH_INSTALL_DIR:-$HOME/.local/bin}"
BINARY="$INSTALL_DIR/ldash"

if [[ -f "$BINARY" ]]; then
  rm -f "$BINARY"
  echo "Removed $BINARY"
else
  echo "Nothing to remove at $BINARY"
fi

echo "Config in ~/.config/ldash was not removed (delete manually if desired)."
