package utils

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/companion"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
)

// ErrOrgRequired is returned when an org ID is needed but the active account has none set.
var ErrOrgRequired = errors.New("Organization is required")

// ResolveOrgID returns orgID from flagOrgID when non-empty; otherwise the active account's org ID.
func ResolveOrgID(accountStore *config.AccountStore, flagOrgID string) (string, error) {
	if flagOrgID != "" {
		return flagOrgID, nil
	}
	active, err := accountStore.ActiveAccount()
	if err != nil || active == nil {
		return "", ErrOrgRequired
	}
	if active.OrgID == "" {
		return "", ErrOrgRequired
	}
	return active.OrgID, nil
}

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

	// Only switch if the client was started by this CLI
	if currentStatus != nil && currentStatus.Agent != olm.AgentName {
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

// WarnUnmanagedClient tells the user when the running client was started by
// another app (e.g. the desktop app on Windows) rather than this CLI, since
// selections made here are saved for the CLI but don't change that client.
// orgID is the newly selected org, or empty when the account changed.
//
// In companion mode the select commands are blocked before reaching this, so
// it only fires with standalone CLI auth.
func WarnUnmanagedClient(orgID string) {
	client := olm.NewClient("")
	if !client.IsRunning() {
		return
	}

	status, err := client.GetStatus()
	if err != nil || status == nil || status.Agent == olm.AgentName {
		return
	}
	if orgID != "" && status.OrgID == orgID {
		return
	}

	agent := status.Agent
	if agent == "" {
		agent = "another application"
	}

	if orgID != "" && status.OrgID != "" {
		logger.Warning("The running client was started by %s, not this CLI, so it is still connected to organization %s.", agent, status.OrgID)
	} else {
		logger.Warning("The running client was started by %s, not this CLI, so it was not changed.", agent)
	}

	if providerName := companion.PlatformProviderName(); providerName != "" && status.Agent == providerName {
		logger.Info("Switch accounts or organizations in %s to change its connection, or run 'pangolin companion enable' to have the CLI use %s's account and organization.", providerName, providerName)
	}
}
