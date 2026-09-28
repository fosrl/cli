package client

import (
	"errors"
	"reflect"
	"testing"

	"github.com/fosrl/cli/internal/api"
)

func TestResolveSavedExitNode(t *testing.T) {
	const savedResourceID = 12
	lister := func(gws ...api.SiteResource) func(string) ([]api.SiteResource, error) {
		return func(string) ([]api.SiteResource, error) { return gws, nil }
	}

	tests := []struct {
		name     string
		resource int
		org      string
		list     func(string) ([]api.SiteResource, error)
		want     []int
		wantID   int
	}{
		{"nothing saved", 0, "org1", lister(), nil, 0},
		{"uses the resource's current sites", savedResourceID, "org1",
			lister(api.SiteResource{SiteResourceID: 11, NiceID: "gw-b", Enabled: true, SiteIDs: []int{1, 2}}, api.SiteResource{SiteResourceID: 12, NiceID: "gw-a", Enabled: true, SiteIDs: []int{2, 3}}), []int{2, 3}, 12},
		{"resource deleted", savedResourceID, "org1", lister(api.SiteResource{SiteResourceID: 11, NiceID: "gw-b", Enabled: true, SiteIDs: []int{1, 2}}), nil, 0},
		{"resource disabled", savedResourceID, "org1", lister(api.SiteResource{SiteResourceID: savedResourceID, NiceID: "gw-a", Enabled: false, SiteIDs: []int{1, 2}}), nil, 0},
		{"resource has no sites", savedResourceID, "org1", lister(api.SiteResource{SiteResourceID: savedResourceID, NiceID: "gw-a", Enabled: true}), nil, 0},
		{"lookup fails: connect without it", savedResourceID, "org1", func(string) ([]api.SiteResource, error) { return nil, errors.New("boom") }, nil, 0},
		{"no session to look it up with", savedResourceID, "org1", nil, nil, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, got := resolveSavedExitNode(tt.resource, tt.org, tt.list)
			if !reflect.DeepEqual(got, tt.want) || gotID != tt.wantID {
				t.Fatalf("got %d %v, want %d %v", gotID, got, tt.wantID, tt.want)
			}
		})
	}
}
