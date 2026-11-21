package ssh

type RunOpts struct {
	User          string
	Alias         string // resource alias as typed by the user; used as the ssh(1) destination so ~/.ssh/config Host blocks match
	Hostname      string // connection target resolved by the Pangolin API; applied via -o Hostname=
	Port          int
	PrivateKeyPEM string
	Certificate   string
	SSHPassthrough
}
