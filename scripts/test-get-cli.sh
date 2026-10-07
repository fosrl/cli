#!/bin/sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

# Load the installer functions without running the installation entry point.
eval "$(sed '/^main "\$@"$/d' "$repo_dir/get-cli.sh")"

uname() {
    case "$1" in
        -s) printf '%s' "$test_os" ;;
        -m) printf '%s' "$test_arch" ;;
        *) return 1 ;;
    esac
}

check_platform() {
    test_os=$1
    test_arch=$2
    expected=$3
    actual=$(detect_platform)
    if [ "$actual" != "$expected" ]; then
        printf 'FAIL: %s/%s: expected %s, got %s\n' "$test_os" "$test_arch" "$expected" "$actual" >&2
        exit 1
    fi
    printf 'PASS: %s/%s -> %s\n' "$test_os" "$test_arch" "$actual"
}

check_platform Linux armv6l linux_arm32v6
check_platform Linux armv7l linux_arm32
check_platform Linux aarch64 linux_arm64
check_platform Linux arm64 linux_arm64
check_platform Linux x86_64 linux_amd64
check_platform Linux amd64 linux_amd64
check_platform Linux riscv64 linux_riscv64
check_platform Darwin arm64 darwin_arm64
check_platform Darwin x86_64 darwin_amd64
check_platform FreeBSD amd64 freebsd_amd64
check_platform MINGW64_NT-10.0 x86_64 windows_amd64

test_os=Linux
test_arch=unsupported
if (detect_platform) >/dev/null 2>&1; then
    printf 'FAIL: unsupported architecture was accepted\n' >&2
    exit 1
fi
printf 'PASS: unsupported architecture is rejected\n'
