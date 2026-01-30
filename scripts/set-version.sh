#!/usr/bin/env bash
# Bump the CLI version by replacing the version in internal/version/consts.go

set -e

VERSION_FILE="internal/version/consts.go"

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  echo "Example: $0 0.4.0"
  exit 1
fi

NEW_VERSION="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$REPO_ROOT"

if [ ! -f "$VERSION_FILE" ]; then
  echo "Error: $VERSION_FILE not found"
  exit 1
fi

# Replace the version in consts.go (matches: const Version = "X.Y.Z")
case $(uname) in
  Darwin) sed -i '' "s/const Version = \"[^\"]*\"/const Version = \"$NEW_VERSION\"/" "$VERSION_FILE" ;;
  *)      sed -i "s/const Version = \"[^\"]*\"/const Version = \"$NEW_VERSION\"/" "$VERSION_FILE" ;;
esac

if grep -q "const Version = \"$NEW_VERSION\"" "$VERSION_FILE"; then
  echo "Version bumped to $NEW_VERSION in $VERSION_FILE"
else
  echo "Error: failed to update version"
  exit 1
fi
