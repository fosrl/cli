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

// ServiceSocketPath is the named pipe the CLI's own Windows client service
// (`pangolin service install client` / `pangolin up client`) listens on. It is
// distinct from defaultSocketPath so the service can't collide with the
// desktop app's olm instance on the same machine.
const ServiceSocketPath = `\\.\pipe\pangolin-client`

// CompanionAgentName is the OLM agent identifier used by the Pangolin Windows Client.
// Must match the Agent value in the Windows app's OLM tunnel config.
const CompanionAgentName = "Pangolin Windows"

func getDefaultSocketPath() string {
	return defaultSocketPath
}

// GetDefaultSocketPath returns the default socket path (exported for use in other packages)
func GetDefaultSocketPath() string {
	return getDefaultSocketPath()
}

// resolveClientSocketPath picks which pipe a client should talk to when none
// was given explicitly: the desktop app's pipe if something is listening on
// it, otherwise the CLI service's pipe if that is up, otherwise the default
// (so error messages still name the expected pipe).
func resolveClientSocketPath() string {
	if socketExists(defaultSocketPath) {
		return defaultSocketPath
	}
	if socketExists(ServiceSocketPath) {
		return ServiceSocketPath
	}
	return defaultSocketPath
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
