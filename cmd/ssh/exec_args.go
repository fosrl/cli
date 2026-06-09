package ssh

import "strconv"

// buildExecSSHArgs assembles argv for the system ssh(1) binary:
//
//	ssh <identity: -l -i -o Certificate -p> <user OpenSSH options> <hostname> <remote command>...
func buildExecSSHArgs(sshPath, user, hostname string, port int, keyPath, certPath string, pass SSHPassthrough) []string {
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
	// Prefer JIT cert/publickey auth first, but allow interactive fallback
	// (password/keyboard-interactive) when the server supports it.
	args = append(args,
		"-o", "PubkeyAuthentication=yes",
		"-o", "PreferredAuthentications=publickey,keyboard-interactive,password",
		"-o", "IdentitiesOnly=yes",
		"-o", "PasswordAuthentication=yes",
		"-o", "KbdInteractiveAuthentication=yes",
	)
	// The built-in SSH server generates a fresh ephemeral host key on every
	// restart, so skip known_hosts checking to avoid spurious MITM warnings.
	// LogLevel=ERROR suppresses the "Permanently added ..." informational line.
	args = append(args, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=ERROR")
	if port > 0 {
		args = append(args, "-p", strconv.Itoa(port))
	}
	args = append(args, pass.Options...)
	args = append(args, hostname)
	args = append(args, pass.RemoteCommand...)
	return args
}
