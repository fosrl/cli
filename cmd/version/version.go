package version

import (
	"fmt"

	"github.com/fosrl/cli/internal/version"
	"github.com/spf13/cobra"
)

var VersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  "Print the version number of Pangolin CLI",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(version.Version)
	},
}

