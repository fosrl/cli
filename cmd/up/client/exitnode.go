package client

import (
	"errors"
	"fmt"

	"github.com/fosrl/cli/internal/api"
)

// resolveExitNodeByNiceID returns the resource ID and current site IDs of the
// exit node with the given niceId, as passed to --exit-node. Unlike a saved
// exit node, this one was asked for explicitly, so failing to resolve it is an
// error rather than a reason to connect without one.
//
// list is nil when there's no user session to query the server with.
func resolveExitNodeByNiceID(niceID string, orgID string, list func(orgID string) ([]api.SiteResource, error)) (int, []int, error) {
	if list == nil || orgID == "" {
		return 0, nil, errors.New("--exit-node needs a logged-in session to look up; pass --exit-node-site-ids and --exit-node-resource-id instead")
	}

	gateways, err := list(orgID)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to look up exit node '%s': %w", niceID, err)
	}

	for _, g := range gateways {
		if g.NiceID != niceID {
			continue
		}
		if !g.Enabled || len(g.SiteIDs) == 0 {
			return 0, nil, fmt.Errorf("exit node '%s' is disabled or has no sites", niceID)
		}
		return g.SiteResourceID, g.SiteIDs, nil
	}

	return 0, nil, fmt.Errorf("exit node '%s' not found", niceID)
}
