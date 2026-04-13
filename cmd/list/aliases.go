package list

import (
	"fmt"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

const aliasesPageSize = 1000

func aliasesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "aliases",
		Short: "Print every private host alias you can reach in the current organization",
		Long:  `Lists all private site aliases you have access to in your selected organization—one name per line.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			apiClient := api.FromContext(cmd.Context())
			accountStore := config.AccountStoreFromContext(cmd.Context())

			orgID, err := utils.ResolveOrgID(accountStore, "")
			if err != nil {
				return err
			}

			for page := 1; ; page++ {
				data, err := apiClient.ListUserResourceAliases(orgID, page, aliasesPageSize)
				if err != nil {
					return fmt.Errorf("list user resource aliases: %w", err)
				}
				for _, a := range data.Aliases {
					fmt.Println(a)
				}
				if len(data.Aliases) == 0 {
					break
				}
				if len(data.Aliases) < aliasesPageSize {
					break
				}
			}
			return nil
		},
	}
}
