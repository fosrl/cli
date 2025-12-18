package status

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

var flagJSON bool

var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Show client status",
	Long:  "Display current client connection status and peer information",
	Run: func(cmd *cobra.Command, args []string) {
		// Get socket path from config or use default
		client := olm.NewClient("")

		// Check if client is running
		if !client.IsRunning() {
			logger.Info("No client is currently running")
			return
		}

		// Get status
		status, err := client.GetStatus()
		if err != nil {
			logger.Error("Error: %v", err)
			os.Exit(1)
		}

		// Print raw JSON if flag is set, otherwise print formatted table
		if flagJSON {
			printJSON(status)
		} else {
			printStatusTable(status)
		}
	},
}

// addStatusClientFlags adds all status client flags to the given command
func addStatusClientFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&flagJSON, "json", false, "Print raw JSON response")
}

func init() {
	addStatusClientFlags(ClientCmd)
}

// printJSON prints the status response as JSON
func printJSON(status *olm.StatusResponse) {
	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		logger.Error("Error marshaling JSON: %v", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
}

// printStatusTable prints the status information in a table format
func printStatusTable(status *olm.StatusResponse) {
	// Print connection status
	headers := []string{"AGENT", "VERSION", "STATUS", "ORG"}
	rows := [][]string{
		{
			status.Agent,
			status.Version,
			formatStatus(status.Connected),
			status.OrgID,
		},
	}
	utils.PrintTable(headers, rows)

	// Print peers if there are any
	if len(status.PeerStatuses) > 0 {
		fmt.Println("")
		peerHeaders := []string{"SITE", "ENDPOINT", "STATUS", "LAST SEEN", "RELAY"}
		peerRows := [][]string{}

		for _, peer := range status.PeerStatuses {
			lastSeen := formatLastSeen(peer.LastSeen.Format(time.RFC3339))

			peerRows = append(peerRows, []string{
				peer.SiteName,
				peer.Endpoint,
				formatStatus(peer.Connected),
				lastSeen,
				fmt.Sprintf("%t", peer.IsRelay),
			})

		}
		utils.PrintTable(peerHeaders, peerRows)
	} else {
		fmt.Println("\nNo peers connected")
	}
}

// formatStatus formats the connection status
func formatStatus(connected bool) string {
	if connected {
		return "Connected"
	}
	return "Disconnected"
}

// formatLastSeen formats the last seen timestamp
func formatLastSeen(lastSeenStr string) string {
	if lastSeenStr == "" {
		return "-"
	}

	// Parse the timestamp
	t, err := time.Parse(time.RFC3339, lastSeenStr)
	if err != nil {
		return lastSeenStr // Return as-is if parsing fails
	}

	// Format as relative time if recent, otherwise absolute
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return fmt.Sprintf("%.0fs ago", diff.Seconds())
	} else if diff < time.Hour {
		return fmt.Sprintf("%.0fm ago", diff.Minutes())
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%.1fh ago", diff.Hours())
	} else {
		return t.Format("2006-01-02 15:04:05")
	}
}
