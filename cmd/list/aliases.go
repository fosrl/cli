package list

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/utils"
	"github.com/spf13/cobra"
)

const aliasesPageSize = 1000

type aliasesFetchOptions struct {
	includeLabels bool
	labelFilter   []string
	status        string
}

func aliasesCmd() *cobra.Command {
	var withLabels bool
	var labelFilter []string

	cmd := &cobra.Command{
		Use:   "aliases",
		Short: "Print every private host alias you can reach in the current organization",
		Long:  `Lists all private site aliases you have access to in your selected organization—one name per line.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("labels") {
				withLabels = true
			}

			apiClient := api.FromContext(cmd.Context())
			accountStore := config.AccountStoreFromContext(cmd.Context())

			orgID, err := utils.ResolveOrgID(accountStore, "")
			if err != nil {
				return err
			}

			requested := aliasesFetchOptions{
				includeLabels: withLabels,
				labelFilter:   labelFilter,
				status:        "approved",
			}

			data, err := fetchAllAliases(apiClient, orgID, requested)
			if err != nil {
				return err
			}

			printAliases(data, withLabels)
			return nil
		},
	}

	cmd.Flags().StringSliceVarP(&labelFilter, "label", "l", nil, "Only list aliases for resources with this label (repeatable, OR)")
	cmd.Flags().BoolVarP(&withLabels, "with-labels", "L", false, "Include label names in output (alias and comma-separated labels, tab-separated)")
	_ = cmd.Flags().Bool("labels", false, "Deprecated: use --with-labels")
	_ = cmd.Flags().MarkDeprecated("labels", "use --with-labels instead")

	return cmd
}

func fetchAllAliases(apiClient *api.Client, orgID string, requested aliasesFetchOptions) (*api.ListUserResourceAliasesData, error) {
	effective, clientSideFilter, err := resolveAliasesFetchOptions(apiClient, orgID, requested)
	if err != nil {
		return nil, err
	}

	var combined api.ListUserResourceAliasesData
	for page := 1; ; page++ {
		pageData, err := apiClient.ListUserResourceAliases(orgID, page, aliasesPageSize, listAliasesAPIOptions(effective))
		if err != nil {
			return nil, err
		}

		combined.Aliases = append(combined.Aliases, pageData.Aliases...)
		combined.Items = append(combined.Items, pageData.Items...)
		combined.Pagination = pageData.Pagination

		if len(pageData.Aliases) == 0 {
			break
		}
		if len(pageData.Aliases) < aliasesPageSize {
			break
		}
	}

	if clientSideFilter {
		combined = *filterAliasesByLabels(&combined, requested.labelFilter)
	}

	return &combined, nil
}

func resolveAliasesFetchOptions(apiClient *api.Client, orgID string, requested aliasesFetchOptions) (aliasesFetchOptions, bool, error) {
	effective, clientSideFilter, err := resolveAliasesFetchOptionsWithStatus(apiClient, orgID, requested)
	if err == nil || !isBadRequest(err) || requested.status == "" {
		return effective, clientSideFilter, err
	}

	// Older servers reject unknown query params such as status; retry without it.
	withoutStatus := requested
	withoutStatus.status = ""
	return resolveAliasesFetchOptionsWithStatus(apiClient, orgID, withoutStatus)
}

func resolveAliasesFetchOptionsWithStatus(apiClient *api.Client, orgID string, requested aliasesFetchOptions) (aliasesFetchOptions, bool, error) {
	effective := requested

	if err := probeAliasesPage(apiClient, orgID, effective); err == nil {
		return effective, false, nil
	} else if !isBadRequest(err) {
		return effective, false, err
	}

	if len(requested.labelFilter) > 0 {
		effective = aliasesFetchOptions{
			includeLabels: true,
			labelFilter:   nil,
			status:        requested.status,
		}
		if err := probeAliasesPage(apiClient, orgID, effective); err == nil {
			return effective, true, nil
		} else if !isBadRequest(err) {
			return effective, false, err
		}
	}

	effective = aliasesFetchOptions{status: requested.status}
	if err := probeAliasesPage(apiClient, orgID, effective); err != nil {
		return effective, false, err
	}

	return effective, false, nil
}

func listAliasesAPIOptions(opts aliasesFetchOptions) api.ListUserResourceAliasesOptions {
	return api.ListUserResourceAliasesOptions{
		IncludeLabels: opts.includeLabels,
		LabelFilter:   opts.labelFilter,
		Status:        opts.status,
	}
}

func probeAliasesPage(apiClient *api.Client, orgID string, opts aliasesFetchOptions) error {
	_, err := apiClient.ListUserResourceAliases(orgID, 1, 1, listAliasesAPIOptions(opts))
	return err
}

func isBadRequest(err error) bool {
	var apiErr *api.ErrorResponse
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusBadRequest
}

func filterAliasesByLabels(data *api.ListUserResourceAliasesData, labelFilter []string) *api.ListUserResourceAliasesData {
	if len(labelFilter) == 0 {
		return data
	}

	filterSet := make(map[string]struct{}, len(labelFilter))
	for _, label := range labelFilter {
		filterSet[label] = struct{}{}
	}

	if len(data.Items) == 0 {
		return data
	}

	filteredItems := make([]api.UserResourceAliasItem, 0, len(data.Items))
	for _, item := range data.Items {
		if aliasMatchesLabelFilter(item.Labels, filterSet) {
			filteredItems = append(filteredItems, item)
		}
	}

	aliases := make([]string, len(filteredItems))
	for i, item := range filteredItems {
		aliases[i] = item.Alias
	}

	return &api.ListUserResourceAliasesData{
		Aliases:    aliases,
		Items:      filteredItems,
		Pagination: data.Pagination,
	}
}

func aliasMatchesLabelFilter(labels []string, filterSet map[string]struct{}) bool {
	for _, label := range labels {
		if _, ok := filterSet[label]; ok {
			return true
		}
	}
	return false
}

func printAliases(data *api.ListUserResourceAliasesData, withLabels bool) {
	if !withLabels {
		for _, alias := range data.Aliases {
			fmt.Println(alias)
		}
		return
	}

	if len(data.Items) > 0 {
		for _, item := range data.Items {
			fmt.Printf("%s\t%s\n", item.Alias, strings.Join(item.Labels, ","))
		}
		return
	}

	for _, alias := range data.Aliases {
		fmt.Printf("%s\t\n", alias)
	}
}
