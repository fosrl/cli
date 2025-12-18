package version

import (
	"fmt"

	"github.com/fosrl/cli/internal/logger"
	versionpkg "github.com/fosrl/cli/internal/version"
	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  "Print the version number and check for updates",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(versionpkg.Version)

		// Check for updates
		latest, err := versionpkg.CheckForUpdate()
		if err != nil {
			// Silently fail - don't show error to user for update check failures
			return
		}

		if latest != nil {
			logger.Warning("\nA new version is available: %s (current: %s)", latest.TagName, versionpkg.Version)
			if latest.URL != "" {
				logger.Info("Release: %s", latest.URL)
			}
			fmt.Println()
			logger.Info("Run 'pangolin update' to update to the latest version")
		}
	},
}
