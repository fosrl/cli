package scp

import (
	"errors"
	"os"

	sshcmd "github.com/fosrl/cli/cmd/ssh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	errHostnameRequired = errors.New("API did not return a hostname for the connection")
	errNoClientRunning  = errors.New("No client is currently running. Start the client first.")
	errScpOperands      = errors.New("scp requires at least one remote operand; example: pangolin scp ./local-file my-server.internal:/remote/path")
	errNoRemoteOperand  = errors.New("no remote operand found; at least one of source or destination must be a remote path (host:path or user@host:path)")
)

func SCPCmd() *cobra.Command {
	opts := struct {
		ResourceID string
		Username   string
		Port       int
	}{}

	cmd := &cobra.Command{
		Use:   "scp [flags] <source> <destination>",
		Short: "Run scp using just-in-time SSH certificates",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true, // Forward unknown flags to system scp(1)
		},
		Long: `Run scp(1) in the terminal. Generates a key pair and signs it just-in-time, then executes the system OpenSSH scp client.

Use the resource alias or identifier as the host in remote operands, exactly as you would with regular scp.
The resource alias is resolved to the connected hostname transparently.
Examples:
  pangolin scp ./local-file my-server.internal:/remote/path
  pangolin scp my-server.internal:/var/log/syslog ./syslog
  pangolin scp -r ./dir my-server.internal:~/

Set PANGOLIN_SCP_BINARY to the full path of scp(1) to override PATH lookup on all platforms.`,
		PreRunE: func(c *cobra.Command, args []string) error {
			if len(args) < 2 {
				return errScpOperands
			}
			username, resourceID, found := parseSCPRemoteHost(args)
			if !found {
				return errNoRemoteOperand
			}
			opts.Username = username
			opts.ResourceID = resourceID
			return nil
		},
		Run: func(c *cobra.Command, args []string) {
			client := olm.NewClient("")
			if !client.IsRunning() {
				logger.Error("%v", errNoClientRunning)
				os.Exit(1)
			}

			apiClient := api.FromContext(c.Context())
			accountStore := config.AccountStoreFromContext(c.Context())

			// init a jit connection to the site if we need to because we might not be connected
			_, err := client.JITConnectByResourceID(opts.ResourceID)
			if err != nil {
				logger.Warning("%v", err) // keep warning behavior for backward compatibility
			}

			orgID, err := utils.ResolveOrgID(accountStore, "")
			if err != nil {
				logger.Error("%v", err)
				os.Exit(1)
			}

			privPEM, _, cert, signData, err := sshcmd.GenerateAndSignKey(apiClient, orgID, opts.ResourceID, opts.Username)
			if err != nil {
				logger.Error("%v", err)
				os.Exit(1)
			}
			if signData == nil || signData.Hostname == "" {
				logger.Error("%v", errHostnameRequired)
				os.Exit(1)
			}

			siteIDs := []int{}
			if signData.SiteID != 0 {
				siteIDs = append(siteIDs, signData.SiteID)
			}
			for _, id := range signData.SiteIDs {
				if id != 0 {
					siteIDs = append(siteIDs, id)
				}
			}

			if len(siteIDs) > 0 {
				if err := waitForAnySiteConnection(client, siteIDs); err != nil {
					logger.Error("%v", err)
					os.Exit(1)
				}
			}

			pt := sshcmd.ParseOpenSSHPassThrough(args)

			// When the auth daemon is the native SSH server, restrict
			// pass-through options to the subset it actually supports.
			if signData.AuthDaemonMode == "native" {
				var stripped []string
				pt, stripped = sshcmd.FilterForNativeSCPMode(pt)
				if len(stripped) > 0 {
					logger.Warning("The following options are not supported by the native SSH server and were ignored: %s", sshcmd.NativeStrippedWarning(stripped))
				}
			}

			runOpts := RunOpts{
				User:          signData.User,
				Hostname:      signData.Hostname,
				Port:          opts.Port,
				PrivateKeyPEM: privPEM,
				Certificate:   cert,
				ResourceID:    opts.ResourceID,
				Passthrough:   pt,
			}

			exitCode, err := RunExec(runOpts)
			if err != nil {
				logger.Error("%v", err)
				os.Exit(1)
			}
			os.Exit(exitCode)
		},
	}

	cmd.Flags().IntVarP(&opts.Port, "port", "p", 0, "Remote SCP/SSH port (default: 22)")

	return cmd
}
