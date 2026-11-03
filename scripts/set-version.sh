#!/usr/bin/env bash
# Set version in internal/version/consts.go and Makefile
# (the MSI version is passed to `wix build -d Version=...`, see scripts/build-msi.bat)
# Usage: ./set-version.sh <version>
#   version: Version string (e.g., 1.0.3)

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  echo "  version: Version string (e.g., \"1.0.3\")"
  echo ""
  echo "Example:"
  echo "  $0 1.0.3"
  exit 1
fi

VERSION="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
VERSION_FILE="${REPO_ROOT}/internal/version/consts.go"
MAKEFILE_PATH="${REPO_ROOT}/Makefile"

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+(\.[0-9]+)?(-[a-zA-Z0-9]+)?$ ]]; then
  echo "Warning: Version format may be invalid. Expected format: X.Y.Z or X.Y.Z-suffix"
  echo "  Example: 1.0.3 or 1.0.3-beta"
  read -p "Continue anyway? (y/N): " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
  fi
fi

cd "$REPO_ROOT"

if [ ! -f "$VERSION_FILE" ]; then
  echo "Error: $VERSION_FILE not found"
  exit 1
fi

if [ ! -f "$MAKEFILE_PATH" ]; then
  echo "Error: $MAKEFILE_PATH not found"
  exit 1
fi

echo "Setting version to: ${VERSION}"
echo ""

echo "Updating $VERSION_FILE..."
case $(uname) in
  Darwin) sed -i '' "s/var Version = \"[^\"]*\"/var Version = \"$VERSION\"/" "$VERSION_FILE" ;;
  *)      sed -i    "s/var Version = \"[^\"]*\"/var Version = \"$VERSION\"/" "$VERSION_FILE" ;;
esac
if grep -q "var Version = \"$VERSION\"" "$VERSION_FILE"; then
  echo "  OK"
else
  echo "Error: failed to update version in $VERSION_FILE"
  exit 1
fi

echo "Updating $MAKEFILE_PATH..."
case $(uname) in
  Darwin) sed -i '' "s/^VERSION[[:space:]]*?=[[:space:]]*.*/VERSION ?= $VERSION/" "$MAKEFILE_PATH" ;;
  *)      sed -i    "s/^VERSION[[:space:]]*?=[[:space:]]*.*/VERSION ?= $VERSION/" "$MAKEFILE_PATH" ;;
esac
if grep -q "^VERSION[[:space:]]*?=[[:space:]]*$VERSION$" "$MAKEFILE_PATH"; then
  echo "  OK"
else
  echo "Error: failed to update VERSION in $MAKEFILE_PATH"
  exit 1
fi

echo ""
echo "Version updated to $VERSION in internal/version/consts.go and Makefile"
echo ""
echo "Next: build the Windows binary, then: scripts\\build-msi.bat $VERSION"
