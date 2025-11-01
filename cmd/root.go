package cmd

import (
	"fmt"
	"os"

	"github.com/fosrl/cli/cmd/create"
	"github.com/fosrl/cli/cmd/list"
	"github.com/fosrl/cli/cmd/login"
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
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(
		&cfgFile,
		"config",
		"",
		"config file (default is $HOME/.pangolin.yaml)",
	)

	rootCmd.PersistentFlags().String("org", "", "Organization name")
	viper.BindPFlag("org", rootCmd.PersistentFlags().Lookup("org"))

	// Register verb commands
	rootCmd.AddCommand(list.ListCmd)
	rootCmd.AddCommand(create.CreateCmd)
	rootCmd.AddCommand(login.LoginCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("$HOME")
		viper.SetConfigName(".pangolin")
	}
	viper.AutomaticEnv() // read env variables

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
