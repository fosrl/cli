//go:build windows
// +build windows

package ssh

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
)

// execSSHSearchPaths are fallback locations for the ssh executable on Windows.
var execSSHSearchPaths = []string{
	`C:\Windows\System32\OpenSSH\ssh.exe`,
}

func findExecSSHPathWindows() (string, error) {
	if path, err := exec.LookPath("ssh"); err == nil {
		return path, nil
	}
	for _, p := range execSSHSearchPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("ssh executable not found in PATH or in OpenSSH location (C:\\Windows\\System32\\OpenSSH\\ssh.exe)")
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

// RunExec runs an interactive SSH session by executing the system ssh binary.
// On Windows the system SSH has better support (e.g. terminal, agent). Requires ssh to be installed.
// opts.PrivateKeyPEM and opts.Certificate must be set (JIT key + signed cert).
func RunExec(opts RunOpts) (int, error) {
	sshPath, err := findExecSSHPathWindows()
	if err != nil {
		return 1, err
	}

	keyPath, certPath, cleanup, err := writeExecKeyFilesWindows(opts)
	if err != nil {
		return 1, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	argv := buildExecSSHArgsWindows(sshPath, opts.User, opts.Hostname, opts.Port, keyPath, certPath, opts.PassThrough)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return execExitCode(err), nil
	}
	return 0, nil
}

func writeExecKeyFilesWindows(opts RunOpts) (keyPath, certPath string, cleanup func(), err error) {
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

func buildExecSSHArgsWindows(sshPath, user, hostname string, port int, keyPath, certPath string, passThrough []string) []string {
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
