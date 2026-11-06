//go:build windows

package companion

import (
	"fmt"
	"time"

	"github.com/Microsoft/go-winio"
)

const cliSecretsPipePath = `\\.\pipe\pangolin-manager-cli-secrets`

type pipeSecretsClient struct {
	pipePath string
}

func newPipeSecretsClient() SecretsClient {
	return &pipeSecretsClient{pipePath: cliSecretsPipePath}
}

func (c *pipeSecretsClient) GetUserSecrets(userID string) (UserSecrets, error) {
	timeout := 3 * time.Second
	conn, err := winio.DialPipe(c.pipePath, &timeout)
	if err != nil {
		return UserSecrets{}, fmt.Errorf("connect to manager secrets pipe: %w", err)
	}
	defer conn.Close()
	return requestSecrets(conn, userID)
}
