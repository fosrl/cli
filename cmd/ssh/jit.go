package ssh

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/sshkeys"
	"golang.org/x/crypto/ssh"
)

const (
	pollInitialDelay  = 250 * time.Millisecond
	pollStartInterval = 250 * time.Millisecond
	pollBackoffSteps  = 6
)

func validateSignedCert(pubKey, cert string) error {
	cert = strings.TrimSpace(cert)
	if cert == "" {
		return fmt.Errorf("API returned an empty SSH certificate")
	}

	pubParsed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubKey))
	if err != nil {
		return fmt.Errorf("parse generated public key: %w", err)
	}

	certParsed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(cert))
	if err != nil {
		return fmt.Errorf("parse returned certificate: %w", err)
	}

	certKey, ok := certParsed.(*ssh.Certificate)
	if !ok {
		return fmt.Errorf("API returned %q instead of an SSH certificate", certParsed.Type())
	}

	if !bytes.Equal(certKey.Key.Marshal(), pubParsed.Marshal()) {
		return fmt.Errorf("returned certificate does not match generated key")
	}

	return nil
}

// GenerateAndSignKey generates an Ed25519 key pair and signs the public key via the API.
func GenerateAndSignKey(client *api.Client, orgID string, resourceID string, username string) (privPEM, pubKey, cert string, signData *api.SignSSHKeyData, err error) {
	privPEM, pubKey, err = sshkeys.GenerateKeyPair()
	if err != nil {
		return "", "", "", nil, fmt.Errorf("generate key pair: %w", err)
	}

	initResp, err := client.SignSSHKey(orgID, api.SignSSHKeyRequest{
		PublicKey: pubKey,
		Resource:  resourceID,
		Username:  username,
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
		if initResp.AuthDaemonMode != "native" {
			if err := validateSignedCert(pubKey, initResp.Certificate); err != nil {
				return "", "", "", nil, fmt.Errorf("SSH error: invalid certificate: %w", err)
			}
		}
		// return the data as this is okay
		return privPEM, pubKey, initResp.Certificate, initResp, nil
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
				if initResp.AuthDaemonMode != "native" {
					if err := validateSignedCert(pubKey, initResp.Certificate); err != nil {
						return "", "", "", nil, fmt.Errorf("SSH error: invalid certificate: %w", err)
					}
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
