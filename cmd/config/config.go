package configcmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fosrl/cli/internal/config"
	"github.com/spf13/cobra"
)

func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View and edit CLI configuration",
		Long:  `View and edit persistent CLI configuration without manually editing the config file.`,
	}

	cmd.AddCommand(configPathCmd())
	cmd.AddCommand(configListCmd())
	cmd.AddCommand(configGetCmd())
	cmd.AddCommand(configSetCmd())

	return cmd
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
	return &cobra.Command{
		Use:   "get [key]",
		Short: "Get a config value, or dump config when no key is given",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.ConfigFromContext(cmd.Context())

			if len(args) == 0 {
				return dumpConfig(cfg)
			}

			value, err := cfg.GetKey(args[0])
			if err != nil {
				return err
			}
			fmt.Println(value)
			return nil
		},
	}
}

func configSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value",
		Long: `Set a config value and write it to the config file.

Supported keys:
  ` + strings.Join(config.SupportedConfigKeys(), "\n  ") + `

Examples:
  pangolin config set up.tunnel_dns true
  pangolin config set up.upstream_dns 10.0.0.53
  pangolin config set up.upstream_dns 10.0.0.53,10.0.0.54
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.ConfigFromContext(cmd.Context())

			if err := cfg.SetKey(args[0], args[1]); err != nil {
				return err
			}
			if err := cfg.Save(); err != nil {
				return err
			}

			value, err := cfg.GetKey(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("%s = %s\n", args[0], value)
			return nil
		},
	}
}

func dumpConfig(cfg *config.Config) error {
	out := map[string]any{
		"log_level":              cfg.LogLevel,
		"log_file":               cfg.LogFile,
		"disable_update_check":   cfg.DisableUpdateCheck,
		"disable_companion_mode": cfg.DisableCompanionMode,
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
	if len(up) > 0 {
		out["up"] = up
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
