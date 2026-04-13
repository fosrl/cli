package ssh

import (
	"errors"
	"os"
	"runtime"
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
		Exec       bool
		Port       int
	}{}

	cmd := &cobra.Command{
		Use:   "ssh <resource alias or identifier>",
		Short: "Run an interactive SSH session",
		Long:  `Run an SSH client in the terminal. Generates a key pair and signs it just-in-time, then connects to the target resource.`,
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
			
			// logger.Info("signData: %+v", signData)
			
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

			passThrough := args[1:]
			runOpts := RunOpts{
				User:          signData.User,
				Hostname:      signData.Hostname,
				Port:          opts.Port,
				PrivateKeyPEM: privPEM,
				Certificate:   cert,
				PassThrough:   passThrough,
			}

			// On Windows, use the system ssh binary by default (better terminal/agent support).
			useExec := opts.Exec || runtime.GOOS == "windows"
			if len(passThrough) > 0 && !useExec {
				logger.Warning("Passthrough arguments are ignored by the built-in client. Use --exec to pass them to the system ssh.")
			}
			var exitCode int
			if useExec {
				exitCode, err = RunExec(runOpts)
			} else {
				exitCode, err = RunNative(runOpts)
			}
			if err != nil {
				logger.Error("%v", err)
				os.Exit(1)
			}
			os.Exit(exitCode)
		},
	}

	cmd.Flags().BoolVar(&opts.Exec, "exec", false, "Use system ssh binary instead of the built-in client")
	cmd.Flags().IntVarP(&opts.Port, "port", "p", 0, "SSH port (default: 22)")

	cmd.AddCommand(SignCmd())

	return cmd
}
