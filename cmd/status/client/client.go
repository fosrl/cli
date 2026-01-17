package client

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

type ClientStatusCmdOpts = struct {
	JSON bool
}

func ClientStatusCmd() *cobra.Command {
	opts := ClientStatusCmdOpts{}

	cmd := &cobra.Command{
		Use:   "client",
		Short: "Show client status",
		Long:  "Display current client connection status and peer information",
		Run: func(cmd *cobra.Command, args []string) {
			if err := clientStatusMain(&opts); err != nil {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().BoolVar(&opts.JSON, "json", false, "Print raw JSON response")

	return cmd
}

func clientStatusMain(opts *ClientStatusCmdOpts) error {
	// Get socket path from config or use default
	client := olm.NewClient("")

	// Check if client is running
	if !client.IsRunning() {
		logger.Info("No client is currently running")
		return nil
	}

	// Get status
	status, err := client.GetStatus()
	if err != nil {
		logger.Error("Error: %v", err)
		return err
	}

	// Print raw JSON if flag is set, otherwise print formatted table
	if opts.JSON {
		return printJSON(status)
	} else {
		printStatusTable(status)
	}

	return nil
}

// printJSON prints the status response as JSON
func printJSON(status *olm.StatusResponse) error {
	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		logger.Error("Error marshaling JSON: %v", err)
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

// printStatusTable prints the status information in a table format
func printStatusTable(status *olm.StatusResponse) {
	// Print connection status
	headers := []string{"AGENT", "VERSION", "STATUS", "ORG"}
	rows := [][]string{
		{
			status.Agent,
			status.Version,
			formatStatus(status.Connected, status.Registered),
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
				formatStatus(peer.Connected, true), // Peers don't have registered field, use true
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
// Status is only "Connected" when both connected and registered are true
func formatStatus(connected, registered bool) string {
	if connected && registered {
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
