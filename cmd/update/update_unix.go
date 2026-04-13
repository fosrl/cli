//go:build !windows

package update

import (
	"os"
	"os/exec"

	"github.com/fosrl/cli/internal/logger"
	"github.com/spf13/cobra"
)

func UpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update Pangolin CLI to the latest version",
		Long:  "Update Pangolin CLI to the latest version by downloading and running the installation script",
		Run: func(cmd *cobra.Command, args []string) {
			if err := updateMain(); err != nil {
				os.Exit(1)
			}
		},
	}

	return cmd
}

func updateMain() error {
	logger.Info("Updating Pangolin CLI...")

	// Get the path of the current running binary
	execPath, err := os.Executable()
	if err != nil {
		logger.Error("Failed to get current executable path: %v", err)
		return err
	}
	
	logger.Debug("Current executable path: %s", execPath)

	// Execute: curl -fsSL https://static.pangolin.net/get-cli.sh | bash -s -- --path /current/binary/path
	updateCmd := exec.Command("sh", "-c", "curl -fsSL https://static.pangolin.net/get-cli.sh | bash -s -- --path "+execPath)
	updateCmd.Stdin = os.Stdin
	updateCmd.Stdout = os.Stdout
	updateCmd.Stderr = os.Stderr

	if err := updateCmd.Run(); err != nil {
		logger.Error("Failed to update Pangolin CLI: %v", err)
		return err
	}

	logger.Success("Pangolin CLI updated successfully!")

	return nil
}
