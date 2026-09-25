package client

import (
	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
)

// resolveSavedExitNode returns the resource ID and current site IDs of the exit
// node saved by `pangolin select exit-node`, or 0/nil if none should be
// applied. Only the resource's niceId is saved, so its ID and sites always come
// from the server and can't be stale (or break if the niceId was renamed away).
//
// list is nil when there's no user session to query the server with. If the
// saved exit node can't be resolved for any reason, the client connects
// without one and says why.
func resolveSavedExitNode(up config.UpConfig, orgID string, list func(orgID string) ([]api.SiteResource, error)) (int, []int) {
	if up.ExitNodeNiceID == "" {
		return 0, nil
	}

	if orgID != "" && up.ExitNodeOrgID != "" && up.ExitNodeOrgID != orgID {
		logger.Info("Saved exit node '%s' belongs to a different organization; not using it", up.ExitNodeNiceID)
		return 0, nil
	}

	if list == nil || orgID == "" {
		logger.Info("Saved exit node '%s' needs a logged-in session to look up; not using it (pass --exit-node-site-ids to set one explicitly)", up.ExitNodeNiceID)
		return 0, nil
	}

	gateways, err := list(orgID)
	if err != nil {
		logger.Warning("Could not look up saved exit node '%s' (%v); connecting without it", up.ExitNodeNiceID, err)
		return 0, nil
	}

	for _, g := range gateways {
		if g.NiceID != up.ExitNodeNiceID {
			continue
		}
		if !g.Enabled || len(g.SiteIDs) == 0 {
			logger.Warning("Saved exit node '%s' is disabled or has no sites; not using it", g.NiceID)
			return 0, nil
		}
		return g.SiteResourceID, g.SiteIDs
	}

	logger.Warning("Saved exit node '%s' no longer exists; not using it. Run 'pangolin select exit-node' and choose None to forget it", up.ExitNodeNiceID)
	return 0, nil
}
