//go:build !windows

package olm

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"
)

const defaultSocketPath = "/var/run/olm.sock"

func getDefaultSocketPath() string {
	return defaultSocketPath
}

// GetDefaultSocketPath returns the default socket path (exported for use in other packages)
func GetDefaultSocketPath() string {
	return getDefaultSocketPath()
}

// resolveClientSocketPath picks which socket a client should talk to when none
// was given explicitly: the CLI's own if something is listening on it, else
// the desktop app's if that is up, otherwise the default (so error messages
// still name the expected socket). Unlike on Windows the two don't share a
// path, so both can be up; the CLI's goes first so 'pangolin down' can stop it.
func resolveClientSocketPath() string {
	if desktopAppSocketPath == "" || socketListening(defaultSocketPath) {
		return defaultSocketPath
	}
	if socketListening(desktopAppSocketPath) {
		return desktopAppSocketPath
	}
	return defaultSocketPath
}

func newHTTPTransport(socketPath string) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
}

func socketExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// socketListening reports whether something accepts connections on path. A
// stale socket file left by a killed process doesn't count.
func socketListening(path string) bool {
	conn, err := net.DialTimeout("unix", path, time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
