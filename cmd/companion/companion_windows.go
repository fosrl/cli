//go:build windows

package companioncmd

import (
	"fmt"

	"github.com/fosrl/cli/internal/companion"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/spf13/cobra"
)

func CompanionCmd() *cobra.Command {
	clientName := companion.PlatformProviderName()
	short := "Manage companion mode with the Pangolin desktop app"
	long := "Enable or disable companion mode, which uses the Pangolin desktop app for authentication."
	if clientName != "" {
		short = "Manage companion mode with " + clientName
		long = "Enable or disable companion mode, which uses " + clientName + " for authentication."
	}

	cmd := &cobra.Command{
		Use:   "companion",
		Short: short,
		Long:  long,
	}

	cmd.AddCommand(companionEnableCmd())
	cmd.AddCommand(companionDisableCmd())
	cmd.AddCommand(companionStatusCmd())

	return cmd
}

func companionEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable companion mode",
		RunE:  companionEnableMain,
	}
}

func companionDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable companion mode",
		RunE:  companionDisableMain,
	}
}

func companionStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show companion mode status",
		RunE:  companionStatusMain,
	}
}

func companionEnableMain(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	cfg.SetCompanionModeEnabled(true)
	if err := cfg.Save(); err != nil {
		return err
	}

	logger.Success("Companion mode enabled")
	logger.Info("This takes effect on the next pangolin command.")
	if companion.RequiredDesktopAppVersion() != "" {
		clientName := companion.PlatformProviderName()
		logger.Info("Companion mode requires %s version %s or later.", clientName, companion.RequiredDesktopAppVersion())
	}
	return nil
}

func companionDisableMain(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	cfg.SetCompanionModeEnabled(false)
	if err := cfg.Save(); err != nil {
		return err
	}

	logger.Success("Companion mode disabled")
	logger.Info("This takes effect on the next pangolin command.")
	return nil
}

func companionStatusMain(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	report := companion.EvaluateStatus(cfg)
	printStatusReport(report)
	return nil
}

func printStatusReport(report companion.StatusReport) {
	fmt.Printf("Companion mode: %s\n", report.ConfigState)

	switch report.ConfigState {
	case "disabled":
		fmt.Println("Auth source: standalone CLI")
		return
	case "unavailable":
		fmt.Printf("Client: %s\n", report.ClientName)
		fmt.Println("Auth source: standalone CLI")
		return
	}

	fmt.Printf("Client: %s\n", report.ClientName)
	if report.Ready {
		fmt.Println("Ready: yes")
		return
	}

	fmt.Println("Ready: no")
	if report.Suggestion != "" {
		fmt.Println(report.Suggestion)
	}
}
