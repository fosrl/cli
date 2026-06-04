//go:build !windows
// +build !windows

package scp

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

// execSCPSearchPaths are fallback locations for the scp executable when not in PATH.
var execSCPSearchPaths = []string{
	"/usr/bin/scp",
	"/usr/local/bin/scp",
	`C:\\Windows\\System32\\OpenSSH\\scp.exe`,
}

func findExecSCPPath() (string, error) {
	if p, ok := scpBinaryFromEnv(); ok {
		if isExecutable(p) {
			return p, nil
		}
		return "", fmt.Errorf("%s=%q: not an executable file", envSCPBinary, p)
	}
	if path, err := exec.LookPath("scp"); err == nil {
		return path, nil
	}
	for _, p := range execSCPSearchPaths {
		if isExecutable(p) {
			return p, nil
		}
	}
	return "", errors.New("scp executable not found in PATH or in common locations")
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
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

// RunExec runs scp via the system scp binary. opts.PrivateKeyPEM and opts.Certificate
// must be set (JIT key + signed cert).
func RunExec(opts RunOpts) (int, error) {
	scpPath, err := findExecSCPPath()
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

	argv := buildExecSCPArgs(scpPath, opts, keyPath, certPath)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return execExitCode(err), nil
	}
	return 0, nil
}

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
