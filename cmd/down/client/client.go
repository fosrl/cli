package client

import (
	"os"

	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/tui"
	"github.com/spf13/cobra"
)

func ClientDownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Stop the client connection",
		Long:  "Stop the currently running client connection",
		Run:   clientDownMain,
	}

	return cmd
}

func clientDownMain(cmd *cobra.Command, args []string) {
	cfg := config.ConfigFromContext(cmd.Context())

	// Get socket path from config or use default
	client := olm.NewClient("")

	// Check if client is running
	if !client.IsRunning() {
		logger.Info("No client is currently running")
		return
	}

	// Check that the client was started by this CLI by verifying the version
	status, err := client.GetStatus()
	if err != nil {
		logger.Error("Failed to get client status: %v", err)
		os.Exit(1)
	}
	if status.Agent != olm.AgentName {
		logger.Error("Client was not started by Pangolin CLI (version: %s)", status.Version)
		logger.Info("Only clients started by this CLI can be stopped using this command")
		os.Exit(1)
	}

	// Send exit signal
	exitResp, err := client.Exit()
	if err != nil {
		logger.Error("Error: %v", err)
		os.Exit(1)
	}

	// Show log preview until process stops
	completed, err := tui.NewLogPreview(tui.LogPreviewConfig{
		LogFile: cfg.LogFile,
		Header:  "Shutting down client...",
		ExitCondition: func(client *olm.Client, status *olm.StatusResponse) (bool, bool) {
			// Exit when process is no longer running (socket doesn't exist)
			if !client.IsRunning() {
				return true, true
			}
			return false, false
		},
		StatusFormatter: func(isRunning bool, status *olm.StatusResponse) string {
			if !isRunning {
				return "Stopped"
			}
			return "Stopping..."
		},
	})
	if err != nil {
		logger.Error("Error: %v", err)
		os.Exit(1)
	}

	if completed {
		logger.Success("Client shutdown completed")
	} else {
		logger.Info("Client shutdown initiated: %s", exitResp.Status)
	}
}
