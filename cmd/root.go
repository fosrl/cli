package cmd

import (
	"os"
	"strings"

	"github.com/fosrl/cli/cmd/auth"
	"github.com/fosrl/cli/cmd/auth/login"
	"github.com/fosrl/cli/cmd/auth/logout"
	"github.com/fosrl/cli/cmd/down"
	"github.com/fosrl/cli/cmd/logs"
	selectcmd "github.com/fosrl/cli/cmd/select"
	"github.com/fosrl/cli/cmd/status"
	"github.com/fosrl/cli/cmd/up"
	"github.com/fosrl/cli/cmd/version"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "pangolin",
	Short: "Pangolin CLI",
}

// Execute is called by main.go
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// GetRootCmd returns the root command for documentation generation
func GetRootCmd() *cobra.Command {
	return rootCmd
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().String("orgId", "", "Organization ID")
	viper.BindPFlag("orgId", rootCmd.PersistentFlags().Lookup("orgId"))

	// Register verb commands
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(selectcmd.SelectCmd)
	rootCmd.AddCommand(up.UpCmd)
	rootCmd.AddCommand(down.DownCmd)
	rootCmd.AddCommand(logs.LogsCmd)
	rootCmd.AddCommand(status.StatusCmd)
	rootCmd.AddCommand(version.VersionCmd)

	// Add login and logout as top-level aliases
	rootCmd.AddCommand(login.LoginCmd)
	rootCmd.AddCommand(logout.LogoutCmd)

	// Hide the completion command
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Get original user's home directory (works with and without sudo)
		homeDir, err := utils.GetOriginalUserHomeDir()
		if err != nil {
			// Fallback to $HOME if we can't determine original user
			viper.AddConfigPath("$HOME")
		} else {
			// Use original user's home directory for config
			viper.AddConfigPath(homeDir)
		}
		viper.SetConfigName(".pangolin")
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv() // read env variables

	// Initialize logger (must be done before any logging)
	utils.InitLogger()

	if err := viper.ReadInConfig(); err != nil {
		// Only warn if it's not a "file not found" error (which is expected for new users)
		if !strings.Contains(err.Error(), "Not Found") {
			utils.Warning("Failed to read config file: %v", err)
		}
	}

	// Initialize API client (always succeeds - may be unauthenticated)
	if err := api.InitGlobalClient(); err != nil {
		// This should never happen, but log it just in case
		utils.Error("Failed to initialize API client: %v", err)
		os.Exit(1)
	}
}
