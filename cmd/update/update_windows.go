//go:build windows

package update

import (
	"os"

	"github.com/fosrl/cli/internal/logger"
	"github.com/spf13/cobra"
)

func UpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update Pangolin CLI to the latest version",
		Long:  "Update Pangolin CLI to the latest version by downloading the new installer from GitHub",
		Run: func(cmd *cobra.Command, args []string) {
			if err := updateMain(); err != nil {
				os.Exit(1)
			}
		},
	}

	return cmd
}

func updateMain() error {
	logger.Info("To update Pangolin CLI on Windows, please download the latest installer from:")
	logger.Info("https://github.com/fosrl/cli/releases")
	logger.Info("")
	logger.Info("Download and run the latest .msi or .exe installer to update to the newest version.")

	return nil
}