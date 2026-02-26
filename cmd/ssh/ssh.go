package ssh

import (
	"errors"
	"os"
	"runtime"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/spf13/cobra"
)

var (
	errHostnameRequired   = errors.New("API did not return a hostname for the connection")
	errResourceIDRequired = errors.New("Resource (alias or identifier) is required")
	errOrgRequired        = errors.New("Organization is required")
	errNoClientRunning    = errors.New("No client is currently running. Start the client first with `pangolin up`")
	errNoClientRunningWindows = errors.New("No client is currently running. Start the client first in the system tray")
)

func SSHCmd() *cobra.Command {
	opts := struct {
		ResourceID string
		Exec       bool
		Port       int
	}{}

	cmd := &cobra.Command{
		Use:   "ssh <resource alias or identifier> [-- passthrough...]",
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
			if runtime.GOOS != "windows" {
				client := olm.NewClient("")
				if !client.IsRunning() {
					logger.Error("%v", errNoClientRunning)
					os.Exit(1)
				}
			} else {
				// check if the named pipe exists by trying to open it. If it doesn't exist, the client is not running.
				pipePath := `\\.\pipe\pangolin-olm`
				pipeFile, err := os.Open(pipePath)
				if err != nil {
					logger.Error("%v", errNoClientRunningWindows)
					os.Exit(1)
				}
				pipeFile.Close()
			}

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
			if signData == nil || signData.Hostname == "" {
				logger.Error("%v", errHostnameRequired)
				os.Exit(1)
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

	cmd.Args = cobra.MinimumNArgs(1)

	cmd.AddCommand(SignCmd())

	return cmd
}
