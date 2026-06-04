//go:build windows
// +build windows

package scp

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/sys/windows"
)

// execSCPSearchPaths are fallback locations for the scp executable on Windows.
var execSCPSearchPaths = []string{
	`C:\\Windows\\System32\\OpenSSH\\scp.exe`,
}

func findExecSCPPathWindows() (string, error) {
	if p, ok := scpBinaryFromEnv(); ok {
		info, err := os.Stat(p)
		if err != nil {
			return "", fmt.Errorf("%s=%q: %w", envSCPBinary, p, err)
		}
		if info.IsDir() {
			return "", fmt.Errorf("%s=%q: is a directory", envSCPBinary, p)
		}
		return p, nil
	}
	if path, err := exec.LookPath("scp"); err == nil {
		return path, nil
	}
	for _, p := range execSCPSearchPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("scp executable not found in PATH or in OpenSSH location (C:\\Windows\\System32\\OpenSSH\\scp.exe)")
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
	scpPath, err := findExecSCPPathWindows()
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

// setWindowsFileOwnerOnly sets the file's ACL so that only the current user has access.
func setWindowsFileOwnerOnly(path string) error {
	var token windows.Token
	proc := windows.CurrentProcess()
	err := windows.OpenProcessToken(proc, windows.TOKEN_QUERY, &token)
	if err != nil {
		return err
	}
	defer token.Close()

	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return err
	}
	userSID := tokenUser.User.Sid

	access := []windows.EXPLICIT_ACCESS{
		{
			AccessPermissions: windows.GENERIC_ALL,
			AccessMode:        windows.SET_ACCESS,
			Inheritance:       windows.NO_INHERITANCE,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeType:  windows.TRUSTEE_IS_USER,
				TrusteeValue: windows.TrusteeValueFromSID(userSID),
			},
		},
	}

	acl, err := windows.ACLFromEntries(access, nil)
	if err != nil {
		return err
	}

	secInfo := windows.SECURITY_INFORMATION(windows.OWNER_SECURITY_INFORMATION | windows.DACL_SECURITY_INFORMATION | windows.PROTECTED_DACL_SECURITY_INFORMATION)
	return windows.SetNamedSecurityInfo(
		path,
		windows.SE_FILE_OBJECT,
		secInfo,
		userSID,
		nil,
		acl,
		nil,
	)
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
	if err := keyFile.Close(); err != nil {
		os.Remove(keyFile.Name())
		return "", "", nil, err
	}

	if err := setWindowsFileOwnerOnly(keyFile.Name()); err != nil {
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
