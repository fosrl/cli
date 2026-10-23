package login

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fosrl/cli/internal/api"
)

func TestLoginCmdRegistersOrgIDFlag(t *testing.T) {
	cmd := LoginCmd()
	if flag := cmd.Flags().Lookup("org-id"); flag == nil {
		t.Fatal("login command is missing org-id flag")
	}
}

func TestResolveOrgForLoginValidOrgIDReturnsOrg(t *testing.T) {
	client, cleanup := newOrgClient(t, []api.Org{{OrgID: "org-a", Name: "Alpha"}})
	defer cleanup()

	orgID, err := resolveOrgForLogin(client, "user-1", "org-a")
	if err != nil {
		t.Fatalf("resolveOrgForLogin returned error: %v", err)
	}
	if orgID != "org-a" {
		t.Fatalf("orgID = %q, want org-a", orgID)
	}
}

func TestResolveOrgForLoginInvalidOrgIDErrorsWithAvailableOrgs(t *testing.T) {
	client, cleanup := newOrgClient(t, []api.Org{
		{OrgID: "org-a", Name: "Alpha"},
		{OrgID: "org-b", Name: "Beta"},
	})
	defer cleanup()

	_, err := resolveOrgForLogin(client, "user-1", "missing")
	if err == nil {
		t.Fatal("resolveOrgForLogin returned nil error for invalid org")
	}
	for _, want := range []string{"missing", "available organizations", "org-a (Alpha)", "org-b (Beta)"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestResolveOrgForLoginZeroOrgsErrorsClearly(t *testing.T) {
	client, cleanup := newOrgClient(t, nil)
	defer cleanup()

	_, err := resolveOrgForLogin(client, "user-1", "missing")
	if err == nil {
		t.Fatal("resolveOrgForLogin returned nil error for zero orgs")
	}
	for _, want := range []string{"missing", "no organizations"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not contain %q", err.Error(), want)
		}
	}
}

func TestOrgExistsForLogin(t *testing.T) {
	client, cleanup := newOrgClient(t, []api.Org{{OrgID: "org-a", Name: "Alpha"}})
	defer cleanup()

	tests := []struct {
		name  string
		orgID string
		want  bool
	}{
		{name: "valid org", orgID: "org-a", want: true},
		{name: "missing org", orgID: "missing", want: false},
		{name: "empty org", orgID: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := orgExistsForLogin(client, "user-1", tt.orgID)
			if err != nil {
				t.Fatalf("orgExistsForLogin returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("orgExistsForLogin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newOrgClient(t *testing.T, orgs []api.Org) (*api.Client, func()) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/user/user-1/orgs" {
			t.Errorf("path = %s, want /user/user-1/orgs", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"success":true,"error":false,"data":{"orgs":%s}}`, mustMarshalOrgs(t, orgs))
	}))

	client, err := api.NewClient(api.ClientConfig{BaseURL: server.URL})
	if err != nil {
		server.Close()
		t.Fatalf("NewClient returned error: %v", err)
	}

	return client, server.Close
}

func mustMarshalOrgs(t *testing.T, orgs []api.Org) string {
	t.Helper()

	b, err := json.Marshal(orgs)
	if err != nil {
		t.Fatalf("failed to marshal orgs: %v", err)
	}

	return string(b)
}
