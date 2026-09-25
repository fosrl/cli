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
		name string
		up   config.UpConfig
		org  string
		list func(string) ([]api.SiteResource, error)
		want []int
	}{
		{"nothing saved", config.UpConfig{}, "org1", lister(), nil},
		{"uses the resource's current sites", saved, "org1",
			lister(api.SiteResource{NiceID: "gw-b", Enabled: true, SiteIDs: []int{1, 2}}, api.SiteResource{NiceID: "gw-a", Enabled: true, SiteIDs: []int{2, 3}}), []int{2, 3}},
		{"resource deleted", saved, "org1", lister(api.SiteResource{NiceID: "gw-b", Enabled: true, SiteIDs: []int{1, 2}}), nil},
		{"same sites but different resource is not a match", saved, "org1", lister(api.SiteResource{NiceID: "gw-other", Enabled: true, SiteIDs: []int{1, 2}}), nil},
		{"resource disabled", saved, "org1", lister(api.SiteResource{NiceID: "gw-a", Enabled: false, SiteIDs: []int{1, 2}}), nil},
		{"resource has no sites", saved, "org1", lister(api.SiteResource{NiceID: "gw-a", Enabled: true}), nil},
		{"different org", saved, "org2", lister(api.SiteResource{NiceID: "gw-a", Enabled: true, SiteIDs: []int{5}}), nil},
		{"lookup fails: connect without it", saved, "org1", func(string) ([]api.SiteResource, error) { return nil, errors.New("boom") }, nil},
		{"no session to look it up with", saved, "org1", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveSavedExitNode(tt.up, tt.org, tt.list); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}
