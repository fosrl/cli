//go:build !windows

package site

import (
	"fmt"
	"os"
	"syscall"
)

// reexec replaces the current process image with a fresh copy of itself,
// preserving all arguments (including "up site ...") and environment
// variables. On success it never returns.
func reexec() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}

// updateRestart returns the restart hook used after a self-update. On Unix
// the updates package execs the new binary itself, so no hook is needed.
func updateRestart() func() error {
	return nil
}
