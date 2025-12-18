package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/fosrl/cli/cmd/auth"
	"github.com/fosrl/cli/cmd/auth/login"
	"github.com/fosrl/cli/cmd/auth/logout"
	"github.com/fosrl/cli/cmd/down"
	"github.com/fosrl/cli/cmd/logs"
	selectcmd "github.com/fosrl/cli/cmd/select"
	"github.com/fosrl/cli/cmd/status"
	"github.com/fosrl/cli/cmd/up"
	"github.com/fosrl/cli/cmd/update"
	"github.com/fosrl/cli/cmd/version"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	versionpkg "github.com/fosrl/cli/internal/version"
	"github.com/spf13/cobra"
)

// Initialize a root Cobra command.
//
// Set initResources to false when generating documentation to avoid
// parsing configuration files and instantiating the API client, among
// other such external resources. This is to avoid depending on external
// state when doing doc generation.
func RootCommand(initResources bool) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "pangolin",
		Short: "Pangolin CLI",
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Skip update check for version and update commands
			// Check both the command name and if it's one of these specific commands
			cmdName := cmd.Name()
			if cmdName == "version" || cmdName == "update" {
				return
			}

			// Check for updates asynchronously
			versionpkg.CheckForUpdateAsync(func(release *versionpkg.GitHubRelease) {
				logger.Warning("A new version is available: %s (current: %s)", release.TagName, versionpkg.Version)
				logger.Info("Run 'pangolin update' to update to the latest version")
				fmt.Println()
			})
		},
	}

	cmd.AddCommand(auth.AuthCmd)
	cmd.AddCommand(selectcmd.SelectCmd)
	cmd.AddCommand(up.UpCmd)
	cmd.AddCommand(down.DownCmd)
	cmd.AddCommand(logs.LogsCmd)
	cmd.AddCommand(status.StatusCmd)
	cmd.AddCommand(update.UpdateCmd)
	cmd.AddCommand(version.VersionCmd)
	cmd.AddCommand(login.LoginCmd)
	cmd.AddCommand(logout.LogoutCmd)

	if !initResources {
		return cmd, nil
	}

	accountStore, err := config.LoadAccountStore()
	if err != nil {
		return nil, err
	}

	var apiBaseURL string
	var sessionToken string

	if activeAccount, _ := accountStore.ActiveAccount(); activeAccount != nil {
		apiBaseURL = activeAccount.Host
		sessionToken = activeAccount.SessionToken
	} else {
		apiBaseURL = ""
		sessionToken = ""
	}

	client, err := api.InitClient(apiBaseURL, sessionToken)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ctx = api.WithAPIClient(ctx, client)
	ctx = config.WithAccountStore(ctx, accountStore)

	cmd.SetContext(ctx)

	return cmd, nil
}

// Execute is called by main.go
func Execute() {
	cmd, err := RootCommand(true)
	if err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
