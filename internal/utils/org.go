package utils

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
)

// SelectOrgForm lists organizations for a user and prompts them to select one.
// It returns the selected org ID and any error.
// If the user has only one organization, it's automatically selected.
func SelectOrgForm(client *api.Client, userID string) (string, error) {
	orgsResp, err := client.ListUserOrgs(userID)
	if err != nil {
		return "", fmt.Errorf("failed to list organizations: %w", err)
	}

	if len(orgsResp.Orgs) == 0 {
		return "", fmt.Errorf("no organizations found for this user")
	}

	if len(orgsResp.Orgs) == 1 {
		// Auto-select if only one org
		selectedOrg := orgsResp.Orgs[0]
		return selectedOrg.OrgID, nil
	}

	// Multiple orgs - let user select
	type OrgOption struct {
		OrgID string
		Label string
	}

	var orgOptions []huh.Option[OrgOption]
	for _, org := range orgsResp.Orgs {
		label := fmt.Sprintf("%s (%s)", org.Name, org.OrgID)
		orgOptions = append(orgOptions, huh.NewOption(label, OrgOption{
			OrgID: org.OrgID,
			Label: label,
		}))
	}

	var selectedOrgOption OrgOption
	orgSelectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[OrgOption]().
				Title("Select an organization").
				Options(orgOptions...).
				Value(&selectedOrgOption),
		),
	)

	if err := orgSelectForm.Run(); err != nil {
		return "", fmt.Errorf("error selecting organization: %w", err)
	}

	return selectedOrgOption.OrgID, nil
}

// SwitchActiveClientOrg checks if the OLM client is running and switches to the new org if so
// It returns true if a switch was attempted (regardless of success)
func SwitchActiveClientOrg(orgID string) bool {
	client := olm.NewClient("")
	if !client.IsRunning() {
		// Client is not running, nothing to do
		return false
	}

	// Get current status to check current orgId
	currentStatus, err := client.GetStatus()
	if err != nil {
		logger.Warning("Failed to get current status: %v", err)
		return false
	}

	// If already on the target org, no need to switch
	if currentStatus != nil && currentStatus.OrgID == orgID {
		return false
	}

	// Client is running, try to switch org
	_, err = client.SwitchOrg(orgID)
	if err != nil {
		logger.Warning("Failed to switch organization in active client: %v", err)
		logger.Warning("The organization has been saved to config, but the active client may still be using the previous organization.")
		return false
	}

	// Switch was sent successfully
	return true
}
