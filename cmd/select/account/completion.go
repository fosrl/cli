package account

import (
	"maps"
	"slices"
	"strings"

	"github.com/fosrl/cli/internal/config"
	"github.com/spf13/cobra"
)

func completeAccountFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	accountStore := config.AccountStoreFromContext(cmd.Context())

	candidateSet := make(map[string]struct{})

	for _, v := range accountStore.AvailableAccounts() {
		if strings.HasPrefix(v.Email, toComplete) {
			candidateSet[v.Email] = struct{}{}
		}
	}

	return slices.Collect(maps.Keys(candidateSet)), cobra.ShellCompDirectiveNoFileComp
}

func completeHostFlag(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	accountStore := config.AccountStoreFromContext(cmd.Context())

	candidateSet := make(map[string]struct{})

	for _, v := range accountStore.AvailableAccounts() {
		if strings.HasPrefix(v.Host, toComplete) {
			candidateSet[v.Host] = struct{}{}
		}
	}

	return slices.Collect(maps.Keys(candidateSet)), cobra.ShellCompDirectiveNoFileComp
}
