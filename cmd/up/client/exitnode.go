package client

import (
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/logger"
)

// resolveSavedExitNode returns the resource ID and current site IDs of the exit
// node saved by `pangolin select exit-node` on the active account, or 0/nil if
// none should be applied. Only the resource ID is saved (scoped to the
// account's org), so its sites always come from the server and can't be stale
// (or break if the resource's niceId was renamed away).
//
// list is nil when there's no user session to query the server with. If the
// saved exit node can't be resolved for any reason, the client connects
// without one and says why.
func resolveSavedExitNode(savedResourceID int, orgID string, list func(orgID string) ([]api.SiteResource, error)) (int, []int) {
	if savedResourceID == 0 {
		return 0, nil
	}

	if list == nil || orgID == "" {
		logger.Info("Saved exit node needs a logged-in session to look up; not using it (pass --exit-node-site-ids to set one explicitly)")
		return 0, nil
	}

	gateways, err := list(orgID)
	if err != nil {
		logger.Warning("Could not look up saved exit node (%v); connecting without it", err)
		return 0, nil
	}

	for _, g := range gateways {
		if g.SiteResourceID != savedResourceID {
			continue
		}
		if !g.Enabled || len(g.SiteIDs) == 0 {
			logger.Warning("Saved exit node '%s' is disabled or has no sites; not using it", g.NiceID)
			return 0, nil
		}
		return g.SiteResourceID, g.SiteIDs
	}

	logger.Warning("Saved exit node no longer exists; not using it. Run 'pangolin select exit-node' and choose None to forget it")
	return 0, nil
}
