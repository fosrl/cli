package client

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
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
	headers := []string{"AGENT", "VERSION", "STATUS", "ORG", "GATEWAY"}
	rows := [][]string{
		{
			status.Agent,
			status.Version,
			formatStatus(status.Connected, status.Registered),
			status.OrgID,
			formatGateway(status),
		},
	}
	utils.PrintTable(headers, rows)

	// Print peers (and the exit node, if connected) if there are any
	if len(status.PeerStatuses) > 0 || status.ExitNode != nil {
		fmt.Println("")
		peerHeaders := []string{"SITE", "ENDPOINT", "STATUS", "LAST SEEN", "CONNECTION", "GATEWAY"}
		peerRows := [][]string{}

		if status.ExitNode != nil {
			lastSeen := formatLastSeen(status.ExitNode.LastSeen.Format(time.RFC3339))
			peerRows = append(peerRows, []string{
				"Pangolin Server",
				status.ExitNode.Endpoint,
				formatStatus(status.ExitNode.Connected, true),
				lastSeen,
				"Direct",
				"-",
			})
		}

		gatewaySites := make(map[int]bool, len(status.GatewaySiteIDs))
		if status.GatewayActive {
			for _, id := range status.GatewaySiteIDs {
				gatewaySites[id] = true
			}
		}

		// Map iteration order is random; sort so the table is stable between runs.
		peers := make([]*olm.OLMPeerStatus, 0, len(status.PeerStatuses))
		for _, peer := range status.PeerStatuses {
			peers = append(peers, peer)
		}
		sort.Slice(peers, func(i, j int) bool { return peers[i].SiteID < peers[j].SiteID })

		for _, peer := range peers {
			lastSeen := formatLastSeen(peer.LastSeen.Format(time.RFC3339))

			peerRows = append(peerRows, []string{
				peer.SiteName,
				peer.Endpoint,
				formatStatus(peer.Connected, true), // Peers don't have registered field, use true
				lastSeen,
				formatConnectionMode(peer.IsLocal, peer.IsRelay),
				formatGatewayMember(gatewaySites[peer.SiteID]),
			})

		}
		utils.PrintTable(peerHeaders, peerRows)
	} else {
		fmt.Println("\nNo peers connected")
	}
}

// formatGateway summarizes whether the client is routing all traffic through a
// gateway (exit node), and which site resource it was selected from.
func formatGateway(status *olm.StatusResponse) string {
	if !status.GatewayActive {
		return "Off"
	}
	if status.GatewaySiteResourceID != 0 {
		return fmt.Sprintf("Active (resource %d)", status.GatewaySiteResourceID)
	}
	return "Active"
}

// formatGatewayMember marks the sites currently in use as the gateway.
func formatGatewayMember(isGateway bool) string {
	if isGateway {
		return "Yes"
	}
	return "-"
}

// formatConnectionMode summarizes how a peer is currently connected. Local and relay are
// mutually exclusive; when neither applies the peer is connected directly to its public
// endpoint.
func formatConnectionMode(isLocal, isRelay bool) string {
	switch {
	case isLocal:
		return "Local"
	case isRelay:
		return "Relay"
	default:
		return "Direct"
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
