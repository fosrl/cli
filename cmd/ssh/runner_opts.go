package ssh

type RunOpts struct {
	User          string
	Hostname      string
	Port          int
	PrivateKeyPEM string
	Certificate   string
	SSHPassthrough
}
