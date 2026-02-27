// +build !windows

package ssh

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/creack/pty"
	"github.com/mattn/go-isatty"
	"golang.org/x/term"
)

// execSSHSearchPaths are fallback locations for the ssh executable when not in PATH.
var execSSHSearchPaths = []string{
	"/usr/bin/ssh",
	"/usr/local/bin/ssh",
	`C:\Windows\System32\OpenSSH\ssh.exe`,
}

func findExecSSHPath() (string, error) {
	if path, err := exec.LookPath("ssh"); err == nil {
		return path, nil
	}
	for _, p := range execSSHSearchPaths {
		if isExecutable(p) {
			return p, nil
		}
	}
	return "", errors.New("ssh executable not found in PATH or in common locations")
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}

func execExitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

// RunOpts is shared by both the exec and native SSH runners.
// PrivateKeyPEM and Certificate are set just-in-time (JIT) before connect; no file paths.
// Port is optional: 0 means use default (22 or whatever is in Hostname); >0 overrides.
type RunOpts struct {
	User          string
	Hostname      string
	Port          int    // optional; 0 = default
	PrivateKeyPEM string // in-memory private key (PEM, OpenSSH format)
	Certificate   string // in-memory certificate from sign-key API
	PassThrough   []string
}

// RunExec runs an interactive SSH session by executing the system ssh binary
// (with a PTY when stdin is a terminal on Unix). Requires ssh to be installed.
// opts.PrivateKeyPEM and opts.Certificate must be set (JIT key + signed cert).
func RunExec(opts RunOpts) (int, error) {
	sshPath, err := findExecSSHPath()
	if err != nil {
		return 1, err
	}

	keyPath, certPath, cleanup, err := writeExecKeyFiles(opts)
	if err != nil {
		return 1, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	argv := buildExecSSHArgs(sshPath, opts.User, opts.Hostname, opts.Port, keyPath, certPath, opts.PassThrough)
	cmd := exec.Command(argv[0], argv[1:]...)

	usePTY := runtime.GOOS != "windows" && isatty.IsTerminal(os.Stdin.Fd())

	if usePTY {
		return runExecWithPTY(cmd)
	}
	return runExecWithoutPTY(cmd)
}

// writeExecKeyFiles writes PrivateKeyPEM and Certificate to temp files for system ssh.
// Returns keyPath, certPath, cleanup func, error.
func writeExecKeyFiles(opts RunOpts) (keyPath, certPath string, cleanup func(), err error) {
	if opts.PrivateKeyPEM == "" {
		return "", "", nil, errors.New("private key required (JIT flow)")
	}
	keyFile, err := os.CreateTemp("", "pangolin-ssh-key-*")
	if err != nil {
		return "", "", nil, err
	}
	if _, err := keyFile.WriteString(opts.PrivateKeyPEM); err != nil {
		keyFile.Close()
		os.Remove(keyFile.Name())
		return "", "", nil, err
	}
	if err := keyFile.Chmod(0o600); err != nil {
		keyFile.Close()
		os.Remove(keyFile.Name())
		return "", "", nil, err
	}
	if err := keyFile.Close(); err != nil {
		os.Remove(keyFile.Name())
		return "", "", nil, err
	}
	keyPath = keyFile.Name()

	if opts.Certificate != "" {
		certFile, err := os.CreateTemp("", "pangolin-ssh-cert-*")
		if err != nil {
			os.Remove(keyPath)
			return "", "", nil, err
		}
		if _, err := certFile.WriteString(opts.Certificate); err != nil {
			certFile.Close()
			os.Remove(certFile.Name())
			os.Remove(keyPath)
			return "", "", nil, err
		}
		if err := certFile.Close(); err != nil {
			os.Remove(certFile.Name())
			os.Remove(keyPath)
			return "", "", nil, err
		}
		certPath = certFile.Name()
	}

	cleanup = func() {
		os.Remove(keyPath)
		if certPath != "" {
			os.Remove(certPath)
		}
	}
	return keyPath, certPath, cleanup, nil
}

func buildExecSSHArgs(sshPath, user, hostname string, port int, keyPath, certPath string, passThrough []string) []string {
	args := []string{sshPath}
	if user != "" {
		args = append(args, "-l", user)
	}
	if keyPath != "" {
		args = append(args, "-i", keyPath)
	}
	if certPath != "" {
		args = append(args, "-o", "CertificateFile="+certPath)
	}
	if port > 0 {
		args = append(args, "-p", strconv.Itoa(port))
	}
	args = append(args, hostname)
	args = append(args, passThrough...)
	return args
}

func runExecWithPTY(cmd *exec.Cmd) (int, error) {
	// Put local terminal in raw mode so Ctrl+C and Tab are sent as bytes to the
	// remote instead of triggering SIGINT or local completion.
	stdinFd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(stdinFd)
	if err != nil {
		return 1, err
	}
	defer func() { _ = term.Restore(stdinFd, oldState) }()

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return 1, err
	}
	defer ptmx.Close()

	// Initial terminal size from our stdin
	if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
		// Non-fatal: continue without initial size
	}

	// Resize PTY on SIGWINCH
	winchCh := make(chan os.Signal, 1)
	signal.Notify(winchCh, syscall.SIGWINCH)
	go func() {
		for range winchCh {
			_ = pty.InheritSize(os.Stdin, ptmx)
		}
	}()
	defer signal.Stop(winchCh)
	// Trigger initial resize (in case InheritSize failed above)
	winchCh <- syscall.SIGWINCH

	// Forward only SIGTERM to the child (e.g. from kill). Ctrl+C is sent as a
	// byte in raw mode and goes through the PTY to the remote.
	forwardCh := make(chan os.Signal, 1)
	signal.Notify(forwardCh, syscall.SIGTERM)
	go func() {
		for sig := range forwardCh {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()
	defer signal.Stop(forwardCh)

	// Copy stdin -> pty and pty -> stdout
	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	_, _ = io.Copy(os.Stdout, ptmx)

	if err := cmd.Wait(); err != nil {
		return execExitCode(err), nil
	}
	return 0, nil
}

func runExecWithoutPTY(cmd *exec.Cmd) (int, error) {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return execExitCode(err), nil
	}
	return 0, nil
}
