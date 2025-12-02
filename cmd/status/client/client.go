package client

import (
	"fmt"
	"os"
	"time"

	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

var ClientCmd = &cobra.Command{
	Use:   "client",
	Short: "Show client status",
	Long:  "Display current client connection status and peer information",
	Run: func(cmd *cobra.Command, args []string) {
		// Get socket path from config or use default
		client := olm.NewClient("")

		// Check if client is running
		if !client.IsRunning() {
			utils.Info("No client is currently running")
			return
		}

		// Get status
		status, err := client.GetStatus()
		if err != nil {
			utils.Error("Error: %v", err)
			os.Exit(1)
		}

		// Print status table
		printStatusTable(status)
	},
}

// printStatusTable prints the status information in a table format
func printStatusTable(status *olm.StatusResponse) {
	// Print connection status
	headers := []string{"CONNECTED", "REGISTERED", "ORG ID"}
	rows := [][]string{
		{
			fmt.Sprintf("%t", status.Connected),
			fmt.Sprintf("%t", status.Registered),
			status.OrgID,
		},
	}
	utils.PrintTable(headers, rows)

	// Print peers if there are any
	if len(status.PeerStatuses) > 0 {
		fmt.Println("")
		peerHeaders := []string{"SITE ID", "ENDPOINT", "STATUS", "RTT", "LAST SEEN", "RELAY"}
		peerRows := [][]string{}

		for _, peer := range status.PeerStatuses {
			rtt := formatRTT(int64(peer.RTT))
			lastSeen := formatLastSeen(peer.LastSeen.Format(time.RFC3339))

			peerRows = append(peerRows, []string{
				fmt.Sprintf("%d", peer.SiteID),
				peer.Endpoint,
				formatStatus("", peer.Connected),
				rtt,
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
func formatStatus(status string, connected bool) string {
	if connected {
		return "Connected"
	}
	if status != "" {
		return fmt.Sprintf("✗ %s", status)
	}
	return "Disconnected"
}

// formatRTT formats the round-trip time in nanoseconds to a human-readable format
func formatRTT(rttNs int64) string {
	if rttNs == 0 {
		return "-"
	}

	// Convert nanoseconds to milliseconds
	rttMs := float64(rttNs) / float64(time.Millisecond)
	if rttMs < 1 {
		return fmt.Sprintf("%.2fμs", float64(rttNs)/float64(time.Microsecond))
	}
	if rttMs < 1000 {
		return fmt.Sprintf("%.2fms", rttMs)
	}
	return fmt.Sprintf("%.2fs", float64(rttNs)/float64(time.Second))
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
