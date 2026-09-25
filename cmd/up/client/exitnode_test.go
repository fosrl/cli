package client

import (
	"errors"
	"reflect"
	"testing"

	"github.com/fosrl/cli/internal/api"
	"github.com/fosrl/cli/internal/config"
)

func TestResolveSavedExitNode(t *testing.T) {
	saved := config.UpConfig{ExitNodeNiceID: "gw-a", ExitNodeOrgID: "org1"}
	lister := func(gws ...api.SiteResource) func(string) ([]api.SiteResource, error) {
		return func(string) ([]api.SiteResource, error) { return gws, nil }
	}

	tests := []struct {
		name   string
		up     config.UpConfig
		org    string
		list   func(string) ([]api.SiteResource, error)
		want   []int
		wantID int
	}{
		{"nothing saved", config.UpConfig{}, "org1", lister(), nil, 0},
		{"uses the resource's current sites", saved, "org1",
			lister(api.SiteResource{SiteResourceID: 11, NiceID: "gw-b", Enabled: true, SiteIDs: []int{1, 2}}, api.SiteResource{SiteResourceID: 12, NiceID: "gw-a", Enabled: true, SiteIDs: []int{2, 3}}), []int{2, 3}, 12},
		{"resource deleted", saved, "org1", lister(api.SiteResource{NiceID: "gw-b", Enabled: true, SiteIDs: []int{1, 2}}), nil, 0},
		{"same sites but different resource is not a match", saved, "org1", lister(api.SiteResource{NiceID: "gw-other", Enabled: true, SiteIDs: []int{1, 2}}), nil, 0},
		{"resource disabled", saved, "org1", lister(api.SiteResource{NiceID: "gw-a", Enabled: false, SiteIDs: []int{1, 2}}), nil, 0},
		{"resource has no sites", saved, "org1", lister(api.SiteResource{NiceID: "gw-a", Enabled: true}), nil, 0},
		{"different org", saved, "org2", lister(api.SiteResource{NiceID: "gw-a", Enabled: true, SiteIDs: []int{5}}), nil, 0},
		{"lookup fails: connect without it", saved, "org1", func(string) ([]api.SiteResource, error) { return nil, errors.New("boom") }, nil, 0},
		{"no session to look it up with", saved, "org1", nil, nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, got := resolveSavedExitNode(tt.up, tt.org, tt.list)
			if !reflect.DeepEqual(got, tt.want) || gotID != tt.wantID {
				t.Fatalf("got %d %v, want %d %v", gotID, got, tt.wantID, tt.want)
			}
		})
	}
}
