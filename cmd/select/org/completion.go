package org

import (
	"fmt"
	"strings"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/spf13/cobra"
)

func completeOrgID(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	apiClient := api.FromContext(cmd.Context())
	accountStore := config.AccountStoreFromContext(cmd.Context())

	activeAccount, err := accountStore.ActiveAccount()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	orgsResponse, err := apiClient.ListUserOrgs(activeAccount.UserID)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	var candidates []string
	for _, org := range orgsResponse.Orgs {
		if strings.HasPrefix(org.OrgID, toComplete) {
			candidates = append(candidates, fmt.Sprintf("%s\t%s", org.OrgID, org.Name))
		}
	}

	return candidates, cobra.ShellCompDirectiveNoFileComp
}
