package scp

import sshcmd "github.com/fosrl/cli/cmd/ssh"

type RunOpts struct {
	User          string
	Hostname      string
	Port          int
	PrivateKeyPEM string
	Certificate   string
	ResourceID    string
	Passthrough   sshcmd.SSHPassthrough
}
