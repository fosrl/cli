package ssh

import (
	"errors"
	"os"
	"time"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	errHostnameRequired       = errors.New("API did not return a hostname for the connection")
	errResourceIDRequired     = errors.New("Resource (alias or identifier) is required; example: pangolin ssh my-server.internal")
	errNoClientRunning        = errors.New("No client is currently running. Start the client first with `pangolin up`")
	errNoClientRunningWindows = errors.New("No client is currently running. Start the client first in the system tray")
)

func SSHCmd() *cobra.Command {
	opts := struct {
		ResourceID string
		Builtin    bool
		Port       int
	}{}

	cmd := &cobra.Command{
		Use:   "ssh <resource alias or identifier>",
		Short: "Run an interactive SSH session",
		FParseErrWhitelist: cobra.FParseErrWhitelist{
			UnknownFlags: true, // -L, -R, and other ssh(1) flags are forwarded to the system OpenSSH client
		},
		Long: `Run an SSH client in the terminal. Generates a key pair and signs it just-in-time, then connects to the target resource.

By default the system OpenSSH client is used on every platform. You can pass the same options as ssh(1) after the resource name (for example port forwards: -L, -R, -D, and -N), then an optional remote command. Example: pangolin ssh <resource> -L 8080:127.0.0.1:80 -N

Set PANGOLIN_SSH_BINARY to the full path of ssh(1) to override PATH lookup on all platforms.`,
		PreRunE: func(c *cobra.Command, args []string) error {
			if len(args) < 1 || args[0] == "" {
				return errResourceIDRequired
			}
			opts.ResourceID = args[0]
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
				logger.Warning("%v", err) // we pass through this warning for backward compatibility with older olm api servers
			}

			orgID, err := utils.ResolveOrgID(accountStore, "")
			if err != nil {
				logger.Error("%v", err)
				os.Exit(1)
			}

			privPEM, _, cert, signData, err := GenerateAndSignKey(apiClient, orgID, opts.ResourceID)
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

			if len(siteIDs) > 0 { // older versions of the server did not send back the site id so we need to check for backward compatibility
				for _, siteID := range siteIDs {
					deadline := time.Now().Add(15 * time.Second)
					connected := false
					for time.Now().Before(deadline) {
						status, err := client.GetStatus()
						if err == nil {
							if peer, ok := status.PeerStatuses[siteID]; ok && peer.Connected {
								connected = true
								// logger.Info("site is connected")
								break
							}
						}
						time.Sleep(500 * time.Millisecond)
					}
					if !connected {
						logger.Error("site %d is not connected; timed out waiting for connection", siteID)
						os.Exit(1)
					}
				}
			}

			passThrough := mergePassThrough(os.Args, opts.ResourceID, args[1:])
			pt := ParseOpenSSHPassThrough(passThrough)
			runOpts := RunOpts{
				User:           signData.User,
				Hostname:       signData.Hostname,
				Port:           opts.Port,
				PrivateKeyPEM:  privPEM,
				Certificate:    cert,
				SSHPassthrough: pt,
			}

			useBuiltin := opts.Builtin
			if len(passThrough) > 0 && useBuiltin {
				logger.Warning("Extra arguments after the resource are ignored by the built-in client (port forwarding, remote commands, and other ssh(1) options). Omit --builtin to use the system OpenSSH client.")
			}
			var exitCode int
			if useBuiltin {
				exitCode, err = RunNative(runOpts)
			} else {
				exitCode, err = RunExec(runOpts)
			}
			if err != nil {
				logger.Error("%v", err)
				os.Exit(1)
			}
			os.Exit(exitCode)
		},
	}

	cmd.Flags().BoolVar(&opts.Builtin, "builtin", false, "Use the built-in SSH client instead of the system OpenSSH binary (interactive shell only)")
	cmd.Flags().IntVarP(&opts.Port, "port", "p", 0, "Remote SSH port (default: 22)")

	cmd.AddCommand(SignCmd())

	return cmd
}
