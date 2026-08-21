package utils

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/logger"
)

const launcherResourcesPageSize = 1000

// FetchAllLauncherResources pages through GET /org/:orgId/launcher/resources
// until fully collected, optionally filtered by a search query.
func FetchAllLauncherResources(client *api.Client, orgID, query string) ([]api.LauncherResource, error) {
	var all []api.LauncherResource
	for page := 1; ; page++ {
		data, err := client.ListLauncherResources(orgID, api.ListLauncherResourcesOptions{
			Query:    query,
			Page:     page,
			PageSize: launcherResourcesPageSize,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list resources: %w", err)
		}
		all = append(all, data.Resources...)
		if len(data.Resources) < launcherResourcesPageSize {
			break
		}
	}
	return all, nil
}

// SelectResourceForm lists AI-gateway-capable (inference mode) launcher
// resources in an org and prompts the user to select one. If there's exactly
// one, it's automatically selected (and logged, since - unlike account/org
// selection - this choice isn't persisted, so the user should see what was
// picked on every run).
func SelectResourceForm(client *api.Client, orgID string) (*api.LauncherResource, error) {
	resources, err := FetchAllLauncherResources(client, orgID, "")
	if err != nil {
		return nil, err
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("no AI gateway resources found in this organization")
	}

	if len(resources) == 1 {
		selected := resources[0]
		logger.Info("Only one resource found, using: %s (%s)", selected.Name, selected.NiceID)
		return &selected, nil
	}

	type resourceOption struct {
		Resource *api.LauncherResource
		Label    string
	}

	var options []huh.Option[resourceOption]
	for i := range resources {
		r := &resources[i]
		label := fmt.Sprintf("%s (%s) - %s", r.Name, r.NiceID, r.AccessDisplay)
		options = append(options, huh.NewOption(label, resourceOption{Resource: r, Label: label}))
	}

	var selectedOption resourceOption
	resourceSelectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[resourceOption]().
				Title("Select a resource").
				Options(options...).
				Value(&selectedOption),
		),
	)

	if err := resourceSelectForm.Run(); err != nil {
		return nil, fmt.Errorf("error selecting resource: %w", err)
	}

	return selectedOption.Resource, nil
}

// accessURLHostMatches reports whether the host portion of accessUrl matches
// identifier (case-insensitive), tolerating identifier being given either as a
// bare domain or as a full URL.
func accessURLHostMatches(accessURL *string, identifier string) bool {
	if accessURL == nil {
		return false
	}

	target := strings.ToLower(strings.TrimSpace(identifier))
	if u, err := url.Parse(target); err == nil && u.Host != "" {
		target = strings.ToLower(u.Host)
	}

	parsed, err := url.Parse(*accessURL)
	if err != nil {
		return false
	}

	return strings.ToLower(parsed.Host) == target
}

// ResolveResourceByIdentifier finds an exact match by niceId or accessUrl host
// among resources matching a server-side fuzzy search for identifier.
func ResolveResourceByIdentifier(client *api.Client, orgID, identifier string) (*api.LauncherResource, error) {
	candidates, err := FetchAllLauncherResources(client, orgID, identifier)
	if err != nil {
		return nil, err
	}

	var exact []api.LauncherResource
	for _, r := range candidates {
		if r.NiceID == identifier || accessURLHostMatches(r.AccessURL, identifier) {
			exact = append(exact, r)
		}
	}

	switch len(exact) {
	case 0:
		return nil, fmt.Errorf("no resource found matching %q", identifier)
	case 1:
		return &exact[0], nil
	default:
		return nil, fmt.Errorf("multiple resources match %q; use a more specific --resource value", identifier)
	}
}
