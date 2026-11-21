package ssh

import "strconv"

// buildExecSSHArgs assembles argv for the system ssh(1) binary:
//
//	ssh <identity: -l -i -o Certificate -p> <user OpenSSH options> <alias|hostname> <remote command>...
//
// When alias is set and differs from the resolved hostname, the alias is used as the
// ssh(1) destination and the resolved hostname is applied with -o Hostname= (command-line
// options take precedence over ssh_config). This lets the user's ~/.ssh/config Host
// blocks for the alias apply (IdentityFile, User, Port, ProxyJump, ...) while the
// connection still goes to the endpoint Pangolin resolved.
//
// When keyPath is empty (manual authentication: the server issued no signed
// certificate), no identity is forced and IdentitiesOnly is not set, so ssh(1) uses
// its normal identity resolution: ~/.ssh/config IdentityFile entries, default key
// paths, and ssh-agent, with password/keyboard-interactive fallback.
func buildExecSSHArgs(sshPath, user, alias, hostname string, port int, keyPath, certPath string, pass SSHPassthrough) []string {
	args := []string{sshPath}
	if user != "" {
		args = append(args, "-l", user)
	}
	target := hostname
	if alias != "" && alias != hostname {
		args = append(args, "-o", "HostName="+hostname)
		target = alias
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
		"-o", "PasswordAuthentication=yes",
		"-o", "KbdInteractiveAuthentication=yes",
	)
	if keyPath != "" {
		// In JIT mode, restrict identities to the signed ephemeral key (plus any
		// explicit -i the user passed) so ssh-agent keys are not offered.
		args = append(args, "-o", "IdentitiesOnly=yes")
	}
	// The built-in SSH server generates a fresh ephemeral host key on every
	// restart, so skip known_hosts checking to avoid spurious MITM warnings.
	// LogLevel=ERROR suppresses the "Permanently added ..." informational line.
	args = append(args, "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "-o", "LogLevel=ERROR")
	if port > 0 {
		args = append(args, "-p", strconv.Itoa(port))
	}
	args = append(args, pass.Options...)
	args = append(args, target)
	args = append(args, pass.RemoteCommand...)
	return args
}
