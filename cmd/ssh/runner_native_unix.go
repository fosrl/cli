//go:build !windows

package ssh

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// setupWindowChangeHandler sets up terminal window size change handling for Unix systems.
// It listens for SIGWINCH signals and updates the SSH session's terminal size accordingly.
func setupWindowChangeHandler(session *ssh.Session, stdinFd int) {
	winchCh := make(chan os.Signal, 1)
	signal.Notify(winchCh, syscall.SIGWINCH)
	go func() {
		for range winchCh {
			if w, h, err := term.GetSize(stdinFd); err == nil {
				_ = session.WindowChange(h, w)
			}
		}
	}()
	defer signal.Stop(winchCh)
	// Trigger initial window size update
	winchCh <- syscall.SIGWINCH
}
