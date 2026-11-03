#!/usr/bin/env bash
# Reset version placeholders in internal/version/consts.go and Makefile
# Usage: ./reset-version.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION_FILE="${REPO_ROOT}/internal/version/consts.go"
MAKEFILE_PATH="${REPO_ROOT}/Makefile"

DEFAULT_CLI_VERSION="version_replaceme"
DEFAULT_MAKEFILE_VERSION="version_replaceme"

cd "$REPO_ROOT"

if [ ! -f "$VERSION_FILE" ]; then
  echo "Error: $VERSION_FILE not found"
  exit 1
fi

if [ ! -f "$MAKEFILE_PATH" ]; then
  echo "Error: $MAKEFILE_PATH not found"
  exit 1
fi

echo "Resetting versions to defaults..."
echo ""

echo "Updating $VERSION_FILE..."
case $(uname) in
  Darwin) sed -i '' "s/var Version = \"[^\"]*\"/var Version = \"$DEFAULT_CLI_VERSION\"/" "$VERSION_FILE" ;;
  *)      sed -i    "s/var Version = \"[^\"]*\"/var Version = \"$DEFAULT_CLI_VERSION\"/" "$VERSION_FILE" ;;
esac
if grep -q "var Version = \"$DEFAULT_CLI_VERSION\"" "$VERSION_FILE"; then
  echo "  OK"
else
  echo "Error: failed to reset version in $VERSION_FILE"
  exit 1
fi

echo "Updating $MAKEFILE_PATH..."
case $(uname) in
  Darwin) sed -i '' "s/^VERSION[[:space:]]*?=[[:space:]]*.*/VERSION ?= $DEFAULT_MAKEFILE_VERSION/" "$MAKEFILE_PATH" ;;
  *)      sed -i    "s/^VERSION[[:space:]]*?=[[:space:]]*.*/VERSION ?= $DEFAULT_MAKEFILE_VERSION/" "$MAKEFILE_PATH" ;;
esac
if grep -q "^VERSION[[:space:]]*?=[[:space:]]*$DEFAULT_MAKEFILE_VERSION$" "$MAKEFILE_PATH"; then
  echo "  OK"
else
  echo "Error: failed to reset VERSION in $MAKEFILE_PATH"
  exit 1
fi

echo ""
echo "Versions reset:"
echo "  internal/version/consts.go -> $DEFAULT_CLI_VERSION"
echo "  Makefile -> $DEFAULT_MAKEFILE_VERSION"
