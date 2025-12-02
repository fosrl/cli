package down

import (
	"os"

	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/tui"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Stop the client connection",
	Long:  "Stop the currently running client connection",
	Run: func(cmd *cobra.Command, args []string) {
		// Get socket path from config or use default
		client := olm.NewClient("")

		// Check if client is running
		if !client.IsRunning() {
			utils.Info("No client is currently running")
			return
		}

		// Get log file path (same as up client)
		logFile := utils.GetDefaultLogPath()

		// Send exit signal
		exitResp, err := client.Exit()
		if err != nil {
			utils.Error("Error: %v", err)
			os.Exit(1)
		}

		// Show log preview until process stops
		completed, err := tui.NewLogPreview(tui.LogPreviewConfig{
			LogFile: logFile,
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
			utils.Error("Error: %v", err)
			os.Exit(1)
		}

		if completed {
			utils.Success("Client shutdown completed")
		} else {
			utils.Info("Client shutdown initiated: %s", exitResp.Status)
		}
	},
}

func init() {
	DownCmd.AddCommand(ClientCmd)
}
