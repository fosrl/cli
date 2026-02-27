package ssh

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

var errKeyFileRequired = errors.New("--key-file is required")

func SignCmd() *cobra.Command {
	opts := struct {
		ResourceID string
		KeyFile    string
		CertFile   string
	}{}

	cmd := &cobra.Command{
		Use:   "sign <resource-id>",
		Short: "Generate and sign an SSH key, then save to files for use with system SSH.",
		Long:  `Generates a key pair, signs the public key, and writes the private key and certificate to files.`,
		PreRunE: func(c *cobra.Command, args []string) error {
			if opts.KeyFile == "" {
				return errKeyFileRequired
			}
			if len(args) < 1 || args[0] == "" {
				return errResourceIDRequired
			}
			opts.ResourceID = args[0]
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			apiClient := api.FromContext(c.Context())
			accountStore := config.AccountStoreFromContext(c.Context())

			orgID, err := ResolveOrgID(accountStore, "")
			if err != nil {
				logger.Error("%v", err)
				os.Exit(1)
			}

			privPEM, _, cert, signData, err := GenerateAndSignKey(apiClient, orgID, opts.ResourceID)
			if err != nil {
				logger.Error("%v", err)
				os.Exit(1)
			}

			keyPath, err := filepath.Abs(opts.KeyFile)
			if err != nil {
				keyPath = opts.KeyFile
			}
			certPath := opts.CertFile
			if certPath == "" {
				certPath = keyPath + "-cert.pub"
			} else {
				certPath, err = filepath.Abs(certPath)
				if err != nil {
					certPath = opts.CertFile
				}
			}

			if err := os.WriteFile(keyPath, []byte(privPEM), 0o600); err != nil {
				logger.Error("write key file: %v", err)
				os.Exit(1)
			}
			if err := os.WriteFile(certPath, []byte(cert), 0o644); err != nil {
				os.Remove(keyPath)
				logger.Error("write certificate file: %v", err)
				os.Exit(1)
			}

			logger.Success("Private key: %s", keyPath)
			logger.Success("Certificate: %s", certPath)
			fmt.Println()

			// Certificate details table
			utils.PrintTable([]string{"Field", "Value"}, signCertTableRows(signData))
			fmt.Println()

			hostname := signData.Hostname
			if hostname == "" {
				hostname = "<hostname>"
			}
			user := signData.User
			if user == "" {
				user = "<user>"
			}
			fmt.Println("Usage with system ssh (scp, tunnels, etc.):")
			fmt.Printf("  ssh -i %q -o CertificateFile=%q %s@%s\n", keyPath, certPath, user, hostname)
			fmt.Printf("  scp -i %q -o CertificateFile=%q ...\n", keyPath, certPath)
		},
	}

	cmd.Flags().StringVar(&opts.KeyFile, "key-file", "", "Path to write the private key (required)")
	cmd.Args = cobra.ExactArgs(1)
	cmd.Flags().StringVar(&opts.CertFile, "cert-file", "", "Path to write the certificate (default: <key-file>-cert.pub)")

	return cmd
}

// signCertTableRows builds table rows for certificate metadata (Key ID, principals, valid after/before, expires in).
func signCertTableRows(d *api.SignSSHKeyData) [][]string {
	if d == nil {
		return nil
	}
	principals := strings.Join(d.ValidPrincipals, ", ")
	if principals == "" {
		principals = "-"
	}
	return [][]string{
		{"Key ID", d.KeyID},
		{"Principals", principals},
		{"Valid after", formatSignDate(d.ValidAfter)},
		{"Valid before", formatSignDate(d.ValidBefore)},
		{"Expires in", formatExpiresIn(d.ExpiresInSeconds)},
	}
}

func formatSignDate(iso string) string {
	if iso == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.Format("Jan 2, 2006 15:04 MST")
}

func formatExpiresIn(seconds int) string {
	if seconds <= 0 {
		return "-"
	}
	d := time.Duration(seconds) * time.Second
	if d >= 24*time.Hour {
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", days)
	}
	if d >= time.Hour {
		h := int(d.Hours())
		if h == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", h)
	}
	if d >= time.Minute {
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", m)
	}
	return fmt.Sprintf("%d seconds", int(d.Seconds()))
}
