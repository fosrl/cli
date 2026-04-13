//go:build !windows

package olm

import (
	"context"
	"net"
	"net/http"
	"os"
)

const defaultSocketPath = "/var/run/olm.sock"

func getDefaultSocketPath() string {
	return defaultSocketPath
}

// GetDefaultSocketPath returns the default socket path (exported for use in other packages)
func GetDefaultSocketPath() string {
	return getDefaultSocketPath()
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