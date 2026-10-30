package configcmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/utils"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View and edit CLI configuration",
		Long:  `View and edit persistent CLI configuration without manually editing the config file.`,
	}

	cmd.AddCommand(configPathCmd())
	cmd.AddCommand(configShowCmd())
	cmd.AddCommand(configListCmd())
	cmd.AddCommand(configGetCmd())
	cmd.AddCommand(configSetCmd())
	cmd.AddCommand(configResetCmd())

	return cmd
}

// activeOrg returns the active account and its selected org, if any.
func activeOrg(cmd *cobra.Command) (*config.Account, string) {
	accountStore := config.AccountStoreFromContext(cmd.Context())
	if accountStore == nil {
		return nil, ""
	}
	account, err := accountStore.ActiveAccount()
	if err != nil {
		return nil, ""
	}
	return account, account.OrgID
}

// fetchServerSettings returns the settings the active org's server manages.
func fetchServerSettings(cmd *cobra.Command, account *config.Account, orgID string) *config.ClientSettings {
	if account == nil || orgID == "" {
		return nil
	}
	accountStore := config.AccountStoreFromContext(cmd.Context())
	apiClient := api.FromContext(cmd.Context())
	return utils.RefreshClientSettings(apiClient.GetMyClientSettings, accountStore, account, orgID)
}

// resolveSetTarget picks the org a value is saved for; "" means global. With
// neither flag it asks when it can, the same choice the desktop apps offer on
// save; otherwise it stays global, as the command always has.
func resolveSetTarget(cmd *cobra.Command, key, flagOrg string, global bool, activeOrgID string) (string, error) {
	if flagOrg != "" || global || !config.IsOrgConfigKey(key) {
		return flagOrg, nil
	}
	if key == "up.exit_node_resource_id" {
		if activeOrgID == "" {
			return "", fmt.Errorf("%s can only be set per organization: pass --org", key)
		}
		return activeOrgID, nil
	}
	if activeOrgID == "" || !isatty.IsTerminal(os.Stdin.Fd()) {
		return "", nil
	}

	target := ""
	err := huh.NewSelect[string]().
		Title("Apply this setting to").
		Options(
			huh.NewOption("All organizations (global)", ""),
			huh.NewOption(fmt.Sprintf("Only %s", activeOrgID), activeOrgID),
		).
		Value(&target).
		Run()
	return target, err
}

func configPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.ConfigFilePath()
			if err != nil {
				return err
			}
			fmt.Println(path)
			return nil
		},
	}
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Print the current config contents",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.ConfigFromContext(cmd.Context())

			if account, orgID := activeOrg(cmd); orgID != "" {
				settings := fetchServerSettings(cmd, account, orgID)
				fmt.Fprintln(os.Stderr, settingsModeMessage(cfg, orgID, settings))
			}
			return dumpConfig(cfg)
		},
	}
}

// settingsModeMessage says which settings apply to the org.
func settingsModeMessage(cfg *config.Config, orgID string, settings *config.ClientSettings) string {
	mode := "uses the global config"
	if org, ok := cfg.Orgs[orgID]; ok && !org.IsEmpty() {
		mode = "has its own config"
	}
	if settings != nil {
		return fmt.Sprintf("Organization %s %s; some settings are managed by its server", orgID, mode)
	}
	return fmt.Sprintf("Organization %s %s", orgID, mode)
}

func configListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List settable config keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			for _, key := range config.ConfigOptions {
				fmt.Fprintln(cmd.OutOrStdout(), key)
			}
			return nil
		},
	}
}

func configGetCmd() *cobra.Command {
	var orgID string

	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.ConfigFromContext(cmd.Context())

			var value string
			var err error
			if orgID != "" {
				value, err = cfg.GetOrgKey(orgID, args[0])
			} else {
				value, err = cfg.GetKey(args[0])
			}
			if err != nil {
				return err
			}
			fmt.Println(value)
			return nil
		},
	}

	cmd.Flags().StringVar(&orgID, "org", "", "Get the value set for this organization only")
	return cmd
}

func configSetCmd() *cobra.Command {
	var orgID string
	var global bool

	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Long: `Set a config value and write it to the config file.

A value can apply to every organization (--global) or to one (--org), which
takes precedence over the global value. Without either flag you're asked when
running interactively with an organization selected; otherwise it's global.
Settings managed by your organization's server take precedence over both.

Supported keys:
  ` + strings.Join(config.SupportedConfigKeys(), "\n  ") + `

Per-organization only:
  up.exit_node_resource_id

Examples:
  pangolin config set up.tunnel_dns true
  pangolin config set up.upstream_dns 10.0.0.53
  pangolin config set up.upstream_dns 10.0.0.53,10.0.0.54
  pangolin config set up.mtu 1400 --org my-org
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.ConfigFromContext(cmd.Context())
			key, value := args[0], args[1]

			account, activeOrgID := activeOrg(cmd)
			target, err := resolveSetTarget(cmd, key, orgID, global, activeOrgID)
			if err != nil {
				return err
			}

			if target != "" {
				err = cfg.SetOrgKey(target, key, value)
			} else {
				err = cfg.SetKey(key, value)
			}
			if err != nil {
				return err
			}
			if err := cfg.Save(); err != nil {
				return err
			}

			var saved string
			if target != "" {
				saved, err = cfg.GetOrgKey(target, key)
			} else {
				saved, err = cfg.GetKey(key)
			}
			if err != nil {
				return err
			}
			if target != "" {
				fmt.Printf("%s = %s (organization %s)\n", key, saved, target)
			} else {
				fmt.Printf("%s = %s\n", key, saved)
			}

			if config.IsOrgConfigKey(key) && fetchServerSettings(cmd, account, activeOrgID).ManagesKey(key) {
				logger.Warning("%s is managed by %s's server, so this value won't apply there", key, activeOrgID)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&orgID, "org", "", "Set the value for this organization only")
	cmd.Flags().BoolVar(&global, "global", false, "Set the value for all organizations")
	cmd.MarkFlagsMutuallyExclusive("org", "global")
	return cmd
}

func configResetCmd() *cobra.Command {
	var orgID string

	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Make an organization use the global config again",
		Long:  `Remove the values set for an organization (the selected one by default), so it uses the global config again.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.ConfigFromContext(cmd.Context())

			if orgID == "" {
				_, orgID = activeOrg(cmd)
			}
			if orgID == "" {
				return fmt.Errorf("no organization selected: pass --org")
			}

			cfg.ResetOrg(orgID)
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Printf("Organization %s now uses the global config\n", orgID)
			return nil
		},
	}

	cmd.Flags().StringVar(&orgID, "org", "", "Organization to reset (default: selected organization)")
	return cmd
}

func dumpConfig(cfg *config.Config) error {
	out := map[string]any{
		"log_level":              cfg.LogLevel,
		"log_file":               cfg.LogFile,
		"disable_update_check":   cfg.DisableUpdateCheck,
		"disable_companion_mode": cfg.DisableCompanionMode,
	}

	if cfg.SessionCookieName != "" {
		out["session_cookie_name"] = cfg.SessionCookieName
	}

	up := map[string]any{}
	if cfg.IsSet("up.tunnel_dns") {
		up["tunnel_dns"] = cfg.GetBool("up.tunnel_dns")
	}
	if cfg.IsSet("up.override_dns") {
		up["override_dns"] = cfg.GetBool("up.override_dns")
	}
	if cfg.IsSet("up.upstream_dns") {
		up["upstream_dns"] = cfg.GetStringSlice("up.upstream_dns")
	}
	if cfg.IsSet("up.prefer_local_routes") {
		up["prefer_local_routes"] = cfg.GetBool("up.prefer_local_routes")
	}
	if cfg.IsSet("up.exit_node_takes_precedence") {
		up["exit_node_takes_precedence"] = cfg.GetBool("up.exit_node_takes_precedence")
	}
	if cfg.Up.MTU != nil {
		up["mtu"] = *cfg.Up.MTU
	}
	if len(up) > 0 {
		out["up"] = up
	}
	if len(cfg.Orgs) > 0 {
		out["orgs"] = cfg.Orgs
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
