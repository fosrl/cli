package ssh

import (
	"os"
	"strings"
)

// envSSHBinary overrides the ssh(1) executable used by RunExec on all platforms when non-empty.
const envSSHBinary = "PANGOLIN_SSH_BINARY"

func sshBinaryFromEnv() (path string, ok bool) {
	p := strings.TrimSpace(os.Getenv(envSSHBinary))
	if p == "" {
		return "", false
	}
	return p, true
}
