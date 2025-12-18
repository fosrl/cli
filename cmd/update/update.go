package update

import (
	"os"
	"os/exec"

	"github.com/fosrl/cli/internal/logger"
	"github.com/spf13/cobra"
)

var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Pangolin CLI to the latest version",
	Long:  "Update Pangolin CLI to the latest version by downloading and running the installation script",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Updating Pangolin CLI...")

		// Execute: curl -fsSL https://pangolin.net/get-cli.sh | bash
		updateCmd := exec.Command("sh", "-c", "curl -fsSL https://static.pangolin.net/get-cli.sh | bash")
		updateCmd.Stdin = os.Stdin
		updateCmd.Stdout = os.Stdout
		updateCmd.Stderr = os.Stderr

		if err := updateCmd.Run(); err != nil {
			logger.Error("Failed to update Pangolin CLI: %v", err)
			os.Exit(1)
		}

		logger.Success("Pangolin CLI updated successfully!")
	},
}
