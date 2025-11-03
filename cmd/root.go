package cmd

import (
	"os"

	"github.com/fosrl/cli/cmd/auth"
	"github.com/fosrl/cli/cmd/auth/login"
	"github.com/fosrl/cli/cmd/auth/logout"
	selectcmd "github.com/fosrl/cli/cmd/select"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "pangolin",
	Short: "Pangolin CLI - manage your sites and clients easily",
	Long:  `Pangolin CLI is an example Go CLI using Cobra and Viper.`,
}

// Execute is called by main.go
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().String("orgId", "", "Organization ID")
	viper.BindPFlag("orgId", rootCmd.PersistentFlags().Lookup("orgId"))

	// Register verb commands
	rootCmd.AddCommand(auth.AuthCmd)
	rootCmd.AddCommand(selectcmd.SelectCmd)
	
	// Add login and logout as top-level aliases
	rootCmd.AddCommand(login.LoginCmd)
	rootCmd.AddCommand(logout.LogoutCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("$HOME")
		viper.SetConfigName(".pangolin")
		viper.SetConfigType("yaml")
	}
	viper.AutomaticEnv() // read env variables

	// Initialize logger (must be done before any logging)
	utils.InitLogger()

	if err := viper.ReadInConfig(); err != nil {
		utils.Warning("Failed to read config file: %v", err)
	}

	// Initialize API client (always succeeds - may be unauthenticated)
	if err := api.InitGlobalClient(); err != nil {
		// This should never happen, but log it just in case
		utils.Error("Failed to initialize API client: %v", err)
		os.Exit(1)
	}
}
