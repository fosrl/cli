package exitnode

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

type ExitNodeCmdOpts struct {
	ExitNode string
}

// disableChoice is the menu value for the "disable gateway" option.
const disableChoice = -1

func ExitNodeCmd() *cobra.Command {
	opts := ExitNodeCmdOpts{}

	cmd := &cobra.Command{
		Use:   "exit-node",
		Short: "Route all traffic through an exit node",
		Long: `List the exit nodes in your organization and select one to route all
tunnel traffic (full tunnel) through the sites backing it.

While an exit node is active, a "None" option is shown to turn it off.
Requires a running client.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := exitNodeMain(cmd, &opts); err != nil {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&opts.ExitNode, "exit-node", "", "Exit node `NICE-ID` to select")

	return cmd
}

func exitNodeMain(cmd *cobra.Command, opts *ExitNodeCmdOpts) error {
	olmClient := olm.NewClient("")
	if !olmClient.IsRunning() {
		err := fmt.Errorf("no client is currently running; start one with 'pangolin up'")
		logger.Error("%v", err)
		return err
	}

	status, err := olmClient.GetStatus()
	if err != nil {
		logger.Error("Failed to get client status: %v", err)
		return err
	}

	cfg := config.ConfigFromContext(cmd.Context())
	apiClient := api.FromContext(cmd.Context())
	accountStore := config.AccountStoreFromContext(cmd.Context())

	orgID, err := utils.ResolveOrgID(accountStore, "")
	if err != nil {
		logger.Error("%v", err)
		return err
	}

	gateways, err := apiClient.ListGatewayResources(orgID)
	if err != nil {
		logger.Error("Failed to list exit nodes: %v", err)
		return err
	}

	usable := gateways[:0]
	for _, g := range gateways {
		if g.Enabled && len(g.SiteIDs) > 0 {
			usable = append(usable, g)
		}
	}
	// A saved exit node can outlive the active one (e.g. its sites weren't
	// connected on startup), so offer to clear it either way.
	hasGateway := status.GatewayActive || len(cfg.Up.GatewaySiteIDs) > 0
	if len(usable) == 0 && !hasGateway {
		err := fmt.Errorf("no exit nodes available in this organization")
		logger.Error("%v", err)
		return err
	}

	choice := disableChoice
	if opts.ExitNode != "" {
		choice = -2
		for i, g := range usable {
			if g.NiceID == opts.ExitNode {
				choice = i
				break
			}
		}
		if choice == -2 {
			err := fmt.Errorf("exit node '%s' not found or not available", opts.ExitNode)
			logger.Error("%v", err)
			return err
		}
	} else {
		choice, err = selectExitNodeForm(usable, status, hasGateway)
		if err != nil {
			logger.Error("%v", err)
			return err
		}
	}

	if choice == disableChoice {
		if status.GatewayActive {
			if _, err := olmClient.DisableGateway(); err != nil {
				logger.Error("Failed to disable exit node: %v", err)
				return err
			}
		}
		saveGateway(cfg, []int{})
		logger.Success("Exit node disabled")
		return nil
	}

	selected := usable[choice]
	if _, err := olmClient.SelectGateway(selected.SiteIDs); err != nil {
		logger.Error("Failed to select exit node: %v", err)
		return err
	}
	saveGateway(cfg, selected.SiteIDs)

	logger.Success("Routing all traffic through exit node: %s", selected.Name)
	return nil
}

// saveGateway persists the exit node so the next `pangolin up` re-applies it.
// A failure is only a warning: the change is already live on the client.
func saveGateway(cfg *config.Config, siteIDs []int) {
	cfg.Up.GatewaySiteIDs = siteIDs
	if err := cfg.Save(); err != nil {
		logger.Warning("Exit node applied but could not be saved for the next start: %v", err)
	}
}

// selectExitNodeForm returns the index of the chosen gateway, or disableChoice.
func selectExitNodeForm(gateways []api.SiteResource, status *olm.StatusResponse, hasGateway bool) (int, error) {
	options := make([]huh.Option[int], 0, len(gateways)+1)
	if hasGateway {
		options = append(options, huh.NewOption("None (disable exit node)", disableChoice))
	}
	for i, g := range gateways {
		label := fmt.Sprintf("%s (%s)", g.Name, g.NiceID)
		if len(g.SiteNames) > 0 {
			label += " - " + strings.Join(g.SiteNames, ", ")
		}
		if status.GatewayActive && sameSites(g.SiteIDs, status.GatewaySiteIDs) {
			label += " [active]"
		}
		options = append(options, huh.NewOption(label, i))
	}

	// huh starts the cursor on the option matching this value, so default to
	// "None" when it's offered; otherwise it would start on the first exit
	// node with "None" scrolled out of view above it.
	selected := 0
	if hasGateway {
		selected = disableChoice
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select an exit node").
				Options(options...).
				Value(&selected),
		),
	)
	if err := form.Run(); err != nil {
		return 0, fmt.Errorf("error selecting exit node: %w", err)
	}

	return selected, nil
}

func sameSites(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[int]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	for _, id := range b {
		if _, ok := set[id]; !ok {
			return false
		}
	}
	return true
}
