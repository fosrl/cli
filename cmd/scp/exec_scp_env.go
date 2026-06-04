package scp

import (
	"os"
	"strings"
)

// envSCPBinary overrides the scp(1) executable used by RunExec on all platforms when non-empty.
const envSCPBinary = "PANGOLIN_SCP_BINARY"

func scpBinaryFromEnv() (path string, ok bool) {
	p := strings.TrimSpace(os.Getenv(envSCPBinary))
	if p == "" {
		return "", false
	}
	return p, true
}
