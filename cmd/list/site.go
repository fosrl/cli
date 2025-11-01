package list

import (
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "List sites in your organization",
	Run: func(c *cobra.Command, args []string) {
		org := viper.GetString("org")
		if org == "" {
			org = "default-org"
		}

		utils.PrintTable(
			[]string{"SITE ID", "NAME", "STATUS"},
			[][]string{
				{"s-123", "Home Lab", "active"},
				{"s-456", "Edge Site", "inactive"},
			},
		)
	},
}

func init() {
	ListCmd.AddCommand(siteCmd)
}

