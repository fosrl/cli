package configcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

// Where the server settings shown came from.
const (
	serverSettingsFetched     = "fetched"      // from the server just now
	serverSettingsLastFetched = "last_fetched" // fetch failed; as last fetched
	serverSettingsNone        = "none"         // not logged in
)

func configEffectiveCmd() *cobra.Command {
	var orgID string
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "effective",
		Short: "Show the settings that apply to an organization and where each comes from",
		Long: `Show, for each per-organization setting, the global value, the organization's
own value, the value its server manages, and the value that applies when
connecting with ` + "`pangolin up`" + `.

The value that applies is the first set of: the server's, the organization's,
the global one, the default. The exit node is the exception: one the server
locks wins, then the one picked with ` + "`pangolin select exit-node`" + `, then the
organization's, then the server's default.

Flags passed to ` + "`pangolin up`" + ` override global and organization values,
but not ones the server manages.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.ConfigFromContext(cmd.Context())

			account, activeOrgID := activeOrg(cmd)
			if orgID == "" {
				orgID = activeOrgID
			}
			if orgID == "" {
				return fmt.Errorf("no organization selected: pass --org")
			}

			var server *config.ClientSettings
			status := serverSettingsNone
			selectedExitNode := 0
			if account != nil {
				selectedExitNode = account.ExitNode(orgID)
				server, status = loadServerSettings(cmd, account, orgID)
			}

			settings := config.EffectiveSettings(cfg, orgID, server, selectedExitNode)
			if jsonOutput {
				return printEffectiveJSON(cmd.OutOrStdout(), orgID, status, settings)
			}
			printEffectiveTable(cmd.OutOrStdout(), orgID, status, settings)
			return nil
		},
	}

	cmd.Flags().StringVar(&orgID, "org", "", "Organization to show (default: selected organization)")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Print as JSON")
	return cmd
}

// loadServerSettings fetches the settings the org's server manages, falling
// back to the ones it last sent.
func loadServerSettings(cmd *cobra.Command, account *config.Account, orgID string) (*config.ClientSettings, string) {
	accountStore, ok := config.LookupAccountStore(cmd.Context())
	if !ok {
		return account.ServerSettings(orgID), serverSettingsLastFetched
	}
	apiClient, ok := api.ClientFromContext(cmd.Context())
	if !ok {
		return account.ServerSettings(orgID), serverSettingsLastFetched
	}

	var fetchErr error
	fetch := func(orgID string) (*config.ClientSettings, error) {
		settings, err := apiClient.GetMyClientSettings(orgID)
		fetchErr = err
		return settings, err
	}
	settings := utils.RefreshClientSettings(fetch, accountStore, account, orgID)
	if fetchErr != nil {
		return settings, serverSettingsLastFetched
	}
	return settings, serverSettingsFetched
}

func printEffectiveTable(w io.Writer, orgID, status string, settings []config.EffectiveSetting) {
	fmt.Fprintf(w, "Organization: %s\n", orgID)
	switch status {
	case serverSettingsFetched:
		fmt.Fprintln(w, "Server settings: fetched from the server just now")
	case serverSettingsLastFetched:
		fmt.Fprintln(w, "Server settings: could not fetch them, showing the ones last fetched")
	default:
		fmt.Fprintln(w, "Server settings: none (not logged in)")
	}
	fmt.Fprintln(w)

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "KEY\tGLOBAL\tORG\tSERVER\tEFFECTIVE\tSOURCE")
	for _, s := range settings {
		global := orDash(s.Global)
		if s.Key == "up.exit_node_resource_id" {
			global = "n/a"
		}
		effective := s.Effective
		if effective == "" {
			effective = "none"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			s.Key, global, orDash(s.Org), orDash(s.Server), effective, s.Source)
	}
	tw.Flush()
}

func printEffectiveJSON(w io.Writer, orgID, status string, settings []config.EffectiveSetting) error {
	data, err := json.MarshalIndent(struct {
		OrgID          string                    `json:"orgId"`
		ServerSettings string                    `json:"serverSettings"`
		Settings       []config.EffectiveSetting `json:"settings"`
	}{orgID, status, settings}, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(w, string(data))
	return nil
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
