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
	if port > 0 {
		args = append(args, "-p", strconv.Itoa(port))
	}
	args = append(args, pass.Options...)
	args = append(args, hostname)
	args = append(args, pass.RemoteCommand...)
	return args
}
