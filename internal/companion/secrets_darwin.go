//go:build darwin

package companion

import (
	"fmt"
	"net"
	"path/filepath"
	"time"
)

// cliSecretsSocketName is the Unix socket Pangolin for macOS listens on, next
// to its accounts.json. The app checks that the caller runs as the same user
// (or root, for the CLI under sudo).
const cliSecretsSocketName = "cli.sock"

const cliSecretsTimeout = 3 * time.Second

// socketSecretsClient asks the running Pangolin macOS app for secrets, which
// it keeps in its own Keychain items.
type socketSecretsClient struct {
	socketPath string
}

func newSocketSecretsClient(dataDir string) SecretsClient {
	return &socketSecretsClient{socketPath: filepath.Join(dataDir, cliSecretsSocketName)}
}

func (c *socketSecretsClient) GetUserSecrets(userID string) (UserSecrets, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, cliSecretsTimeout)
	if err != nil {
		return UserSecrets{}, fmt.Errorf("connect to desktop app secrets socket: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(cliSecretsTimeout))

	return requestSecrets(conn, userID)
}
