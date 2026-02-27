package sshkeys

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"

	"golang.org/x/crypto/ssh"
)

// GenerateKeyPair generates an Ed25519 SSH key pair in memory and returns the
// private and public keys as strings. Nothing is written to disk.
//
// privateKey: PEM-encoded private key (OpenSSH format).
// publicKey:  Authorized-keys style single line (e.g. "ssh-ed25519 AAAA...").
func GenerateKeyPair() (privateKey string, publicKey string, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate key: %w", err)
	}

	// OpenSSH format (-----BEGIN OPENSSH PRIVATE KEY-----)
	block, err := marshalOpenSSHPrivateKey(priv)
	if err != nil {
		return "", "", fmt.Errorf("marshal private key: %w", err)
	}
	privPEM := string(pem.EncodeToMemory(block))

	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return "", "", fmt.Errorf("marshal public key: %w", err)
	}
	pubLine := string(ssh.MarshalAuthorizedKey(sshPub))
	pubLine = string(bytes.TrimSuffix([]byte(pubLine), []byte("\n")))
	publicKey = pubLine

	return privPEM, publicKey, nil
}

// marshalOpenSSHPrivateKey returns a PEM block for the OpenSSH private key format
// (-----BEGIN OPENSSH PRIVATE KEY-----). Uses crypto/rand for check bytes.
func marshalOpenSSHPrivateKey(key ed25519.PrivateKey) (*pem.Block, error) {
	magic := append([]byte("openssh-key-v1"), 0)

	var w struct {
		CipherName   string
		KdfName      string
		KdfOpts      string
		NumKeys      uint32
		PubKey       []byte
		PrivKeyBlock []byte
	}

	ci := make([]byte, 4)
	if _, err := rand.Read(ci); err != nil {
		return nil, err
	}
	checkVal := uint32(ci[0])<<24 | uint32(ci[1])<<16 | uint32(ci[2])<<8 | uint32(ci[3])

	pk1 := struct {
		Check1  uint32
		Check2  uint32
		Keytype string
		Pub     []byte
		Priv    []byte
		Comment string
		Pad     []byte `ssh:"rest"`
	}{
		Check1:  checkVal,
		Check2:  checkVal,
		Keytype: ssh.KeyAlgoED25519,
		Pub:     []byte(key.Public().(ed25519.PublicKey)),
		Priv:    []byte(key),
		Comment: "",
	}

	blockLen := len(ssh.Marshal(pk1))
	padLen := (8 - (blockLen % 8)) % 8
	pk1.Pad = make([]byte, padLen)
	for i := 0; i < padLen; i++ {
		pk1.Pad[i] = byte(i + 1)
	}

	prefix := []byte{0x0, 0x0, 0x0, 0x0b}
	prefix = append(prefix, []byte(ssh.KeyAlgoED25519)...)
	prefix = append(prefix, []byte{0x0, 0x0, 0x0, 0x20}...)

	w.CipherName = "none"
	w.KdfName = "none"
	w.KdfOpts = ""
	w.NumKeys = 1
	w.PubKey = append(prefix, pk1.Pub...)
	w.PrivKeyBlock = ssh.Marshal(pk1)

	magic = append(magic, ssh.Marshal(w)...)

	return &pem.Block{Type: "OPENSSH PRIVATE KEY", Bytes: magic}, nil
}
