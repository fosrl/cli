//go:build linux

package companion

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

// linuxDaemonSocketPath is the Pangolin Linux client daemon's socket.
const linuxDaemonSocketPath = "/run/pangolin-client/manager.sock"

// daemonRequestCLISecrets is the request byte the daemon answers with a
// user's secrets. It is followed by the UID whose secrets are wanted: the
// daemon checks it against the caller's, except for root.
const daemonRequestCLISecrets byte = 'S'

type socketSecretsClient struct {
	socketPath string
}

func newSocketSecretsClient() SecretsClient {
	return &socketSecretsClient{socketPath: linuxDaemonSocketPath}
}

func (c *socketSecretsClient) GetUserSecrets(userID string) (UserSecrets, error) {
	uid, err := secretsUID()
	if err != nil {
		return UserSecrets{}, err
	}

	conn, err := net.DialTimeout("unix", c.socketPath, 3*time.Second)
	if err != nil {
		return UserSecrets{}, fmt.Errorf("connect to the Pangolin client daemon: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write([]byte{daemonRequestCLISecrets}); err != nil {
		return UserSecrets{}, fmt.Errorf("write request: %w", err)
	}
	if err := binary.Write(conn, binary.LittleEndian, uid); err != nil {
		return UserSecrets{}, fmt.Errorf("write uid: %w", err)
	}
	return requestSecrets(conn, userID)
}

// secretsUID is the user whose secrets to ask for: the user who ran sudo
// when this is root under sudo, otherwise the current user.
func secretsUID() (uint32, error) {
	if u := sudoUser(); u != nil {
		uid, err := strconv.ParseUint(u.Uid, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid SUDO_UID %q", u.Uid)
		}
		return uint32(uid), nil
	}
	return uint32(os.Getuid()), nil
}
