package list

import (
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "List clients in your organization",
	Run: func(c *cobra.Command, args []string) {
		org := viper.GetString("org")
		if org == "" {
			org = "default-org"
		}

		utils.PrintTable(
			[]string{"CLIENT ID", "NAME", "STATUS"},
			[][]string{
				{"c-123", "Client 1", "active"},
				{"c-456", "Client 2", "inactive"},
			},
		)
	},
}

func init() {
	ListCmd.AddCommand(clientCmd)
}

