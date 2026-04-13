//go:build windows

package olm

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/Microsoft/go-winio"
)

const defaultSocketPath = `\\.\pipe\pangolin-olm`

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
			timeout := 2 * time.Second
			return winio.DialPipe(socketPath, &timeout)
		},
	}
}

func socketExists(path string) bool {
	timeout := 1 * time.Second
	conn, err := winio.DialPipe(path, &timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}