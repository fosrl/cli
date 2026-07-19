#!/usr/bin/env bash
# Recompute flake.nix's vendorHash using an ephemeral nixos/nix container,
# so no Nix install touches the host system. Requires docker (or podman).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FLAKE="$REPO_ROOT/flake.nix"
RUNTIME="${CONTAINER_RUNTIME:-docker}"

current_hash="$(grep -oP '(?<=vendorHash = ")[^"]+' "$FLAKE")"

cleanup() {
  sed -i "s|vendorHash = pkgs.lib.fakeHash;|vendorHash = \"$current_hash\";|" "$FLAKE"
}
trap cleanup EXIT

sed -i "s|vendorHash = \"$current_hash\";|vendorHash = pkgs.lib.fakeHash;|" "$FLAKE"

output="$("$RUNTIME" run --rm -e NIXPKGS_ALLOW_UNFREE=1 -v "$REPO_ROOT":/src -w /src nixos/nix \
  sh -c "git config --global --add safe.directory /src && nix --extra-experimental-features 'nix-command flakes' build --impure .#pangolin-cli 2>&1" || true)"

new_hash="$(echo "$output" | grep -oP '(?<=got:\s{4})sha256-\S+' || true)"

if [ -z "$new_hash" ]; then
  echo "Could not determine new vendorHash. Full output:" >&2
  echo "$output" >&2
  exit 1
fi

sed -i "s|vendorHash = pkgs.lib.fakeHash;|vendorHash = \"$new_hash\";|" "$FLAKE"
trap - EXIT

echo "vendorHash updated to: $new_hash"
