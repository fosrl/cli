//go:build windows

package companion

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Microsoft/go-winio"
)

const cliSecretsPipePath = `\\.\pipe\pangolin-manager-cli-secrets`

const (
	secretsStatusOK       uint32 = 0
	secretsStatusNotFound uint32 = 1
	secretsStatusError    uint32 = 2
)

// UserSecrets matches the Windows manager secret store JSON shape.
type UserSecrets struct {
	SessionToken string `json:"sessionToken,omitempty"`
	OlmID        string `json:"olmId,omitempty"`
	OlmSecret    string `json:"olmSecret,omitempty"`
}

// SecretsClient fetches user secrets from the Pangolin Manager service.
type SecretsClient interface {
	GetUserSecrets(userID string) (UserSecrets, error)
}

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

	userIDBytes := []byte(userID)
	if err := binary.Write(conn, binary.LittleEndian, uint32(len(userIDBytes))); err != nil {
		return UserSecrets{}, fmt.Errorf("write user id length: %w", err)
	}
	if _, err := conn.Write(userIDBytes); err != nil {
		return UserSecrets{}, fmt.Errorf("write user id: %w", err)
	}

	var status uint32
	if err := binary.Read(conn, binary.LittleEndian, &status); err != nil {
		return UserSecrets{}, fmt.Errorf("read secrets status: %w", err)
	}

	payloadLen, err := readUint32(conn)
	if err != nil {
		return UserSecrets{}, err
	}
	if payloadLen == 0 {
		return UserSecrets{}, fmt.Errorf("empty secrets response payload")
	}

	payload := make([]byte, payloadLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return UserSecrets{}, fmt.Errorf("read secrets payload: %w", err)
	}

	switch status {
	case secretsStatusOK:
		var secrets UserSecrets
		if err := json.Unmarshal(payload, &secrets); err != nil {
			return UserSecrets{}, fmt.Errorf("decode secrets json: %w", err)
		}
		return secrets, nil
	case secretsStatusNotFound:
		return UserSecrets{}, nil
	case secretsStatusError:
		return UserSecrets{}, fmt.Errorf("%s", string(payload))
	default:
		return UserSecrets{}, fmt.Errorf("unexpected secrets status: %d", status)
	}
}

func readUint32(r io.Reader) (uint32, error) {
	var value uint32
	if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
		return 0, fmt.Errorf("read uint32: %w", err)
	}
	return value, nil
}
