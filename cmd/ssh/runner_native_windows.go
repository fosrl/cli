//go:build windows

package ssh

import (
	"golang.org/x/crypto/ssh"
)

// setupWindowChangeHandler is a no-op on Windows.
// Windows does not support SIGWINCH signals for terminal resize detection.
func setupWindowChangeHandler(session *ssh.Session, stdinFd int) {
	// No-op: Windows doesn't have SIGWINCH
}
