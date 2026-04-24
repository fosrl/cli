package ssh

import (
	"fmt"
	"time"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/sshkeys"
)

const (
	pollInitialDelay  = 250 * time.Millisecond
	pollStartInterval = 250 * time.Millisecond
	pollBackoffSteps  = 6
)

// GenerateAndSignKey generates an Ed25519 key pair and signs the public key via the API.
func GenerateAndSignKey(client *api.Client, orgID string, resourceID string) (privPEM, pubKey, cert string, signData *api.SignSSHKeyData, err error) {
	privPEM, pubKey, err = sshkeys.GenerateKeyPair()
	if err != nil {
		return "", "", "", nil, fmt.Errorf("generate key pair: %w", err)
	}

	initResp, err := client.SignSSHKey(orgID, api.SignSSHKeyRequest{
		PublicKey: pubKey,
		Resource:  resourceID,
	})
	if err != nil {
		return "", "", "", nil, fmt.Errorf("SSH error: %w", err)
	}

	// Collect all message IDs to poll (support both single and multiple).
	var messageIDs []int64
	if len(initResp.MessageIDs) > 0 {
		messageIDs = initResp.MessageIDs
	} else if initResp.MessageID != 0 {
		messageIDs = []int64{initResp.MessageID}
	} else {
		return "", "", "", nil, fmt.Errorf("SSH error: API did not return a message ID")
	}

	time.Sleep(pollInitialDelay)

	interval := pollStartInterval
	for i := 0; i <= pollBackoffSteps; i++ {
		for _, messageID := range messageIDs {
			msg, pollErr := client.GetRoundTripMessage(messageID)
			if pollErr != nil {
				return "", "", "", nil, fmt.Errorf("SSH error: poll: %w", pollErr)
			}
			if msg.Complete {
				if msg.Error != nil && *msg.Error != "" {
					return "", "", "", nil, fmt.Errorf("SSH error: %s", *msg.Error)
				}
				return privPEM, pubKey, initResp.Certificate, initResp, nil
			}
		}
		if i < pollBackoffSteps {
			time.Sleep(interval)
			interval *= 2
		}
	}

	return "", "", "", nil, fmt.Errorf("SSH error: timed out waiting for round-trip message")
}
