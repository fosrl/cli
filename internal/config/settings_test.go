package config

import (
	"os"
	"path/filepath"
	"testing"
)

func intPtr(i int) *int { return &i }

func TestEffectiveExitNode(t *testing.T) {
	orgOverride := OrgConfig{ExitNodeResourceID: intPtr(30)}

	tests := []struct {
		name   string
		server *ClientSettings
		saved  int
		org    OrgConfig
		want   int
	}{
		{"nothing set", nil, 0, OrgConfig{}, 0},
		{"server default only", &ClientSettings{ExitNodeSiteResourceID: 10}, 0, OrgConfig{}, 10},
		{"org config beats server default", &ClientSettings{ExitNodeSiteResourceID: 10}, 0, orgOverride, 30},
		{"user selection beats org config and default", &ClientSettings{ExitNodeSiteResourceID: 10}, 20, orgOverride, 20},
		{"locked beats everything", &ClientSettings{ExitNodeSiteResourceID: 10, ExitNodeLocked: true}, 20, orgOverride, 10},
		{"no server settings", nil, 20, orgOverride, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EffectiveExitNode(tt.server, tt.saved, tt.org); got != tt.want {
				t.Errorf("EffectiveExitNode() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestManagesKeyNilSettings(t *testing.T) {
	var settings *ClientSettings
	if settings.ManagesKey("up.mtu") || settings.LockedExitNode() != 0 || settings.UpstreamDNS() != nil {
		t.Error("nil settings should manage nothing")
	}
}

func TestSetKeyRejectsOrgOnlyKeyGlobally(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetKey("up.exit_node_resource_id", "5"); err == nil {
		t.Error("expected the exit node to be rejected as a global setting")
	}
}

func TestOrgConfigRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	if err := os.MkdirAll(filepath.Join(home, ".config", "pangolin"), 0o755); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetKey("up.mtu", "1400"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetOrgKey("my-org", "up.mtu", "1300"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetOrgKey("my-org", "up.tunnel_dns", "false"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetOrgKey("my-org", "up.exit_node_resource_id", "7"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetOrgKey("my-org", "up.mtu", "1"); err == nil {
		t.Error("expected an out of range MTU to be rejected")
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Up.MTU == nil || *loaded.Up.MTU != 1400 {
		t.Errorf("global MTU = %v, want 1400", loaded.Up.MTU)
	}
	org := loaded.Orgs["my-org"]
	if org.MTU == nil || *org.MTU != 1300 {
		t.Errorf("org MTU = %v, want 1300", org.MTU)
	}
	if org.TunnelDNS == nil || *org.TunnelDNS {
		t.Errorf("org tunnel DNS = %v, want an explicit false", org.TunnelDNS)
	}
	if org.ExitNodeResourceID == nil || *org.ExitNodeResourceID != 7 {
		t.Errorf("org exit node = %v, want 7", org.ExitNodeResourceID)
	}

	loaded.ResetOrg("my-org")
	if err := loaded.Save(); err != nil {
		t.Fatal(err)
	}
	reset, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !reset.Orgs["my-org"].IsEmpty() {
		t.Errorf("org should be empty after a reset, got %+v", reset.Orgs["my-org"])
	}
	if reset.Up.MTU == nil || *reset.Up.MTU != 1400 {
		t.Errorf("global MTU should survive a reset, got %v", reset.Up.MTU)
	}
}

func TestServerSettingsRoundTripThroughAccountStore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SUDO_USER", "")
	if err := os.MkdirAll(filepath.Join(home, ".config", "pangolin"), 0o755); err != nil {
		t.Fatal(err)
	}

	store, err := LoadAccountStore(nil)
	if err != nil {
		t.Fatal(err)
	}
	account := Account{UserID: "u1", SessionToken: "t"}
	enabled := false
	account.SetServerSettings("my-org", &ClientSettings{
		OverrideDNS:  &enabled,
		MatchDomains: []string{"*.corp"},
		PrimaryDNS:   "10.0.0.1",
	})
	account.SetExitNode("my-org", 9)
	store.ActiveUserID = "u1"
	store.Accounts["u1"] = account
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadAccountStore(nil)
	if err != nil {
		t.Fatal(err)
	}
	active, err := loaded.ActiveAccount()
	if err != nil {
		t.Fatal(err)
	}
	settings := active.ServerSettings("my-org")
	if settings == nil || settings.OverrideDNS == nil || *settings.OverrideDNS {
		t.Fatalf("explicit false should survive, got %+v", settings)
	}
	if len(settings.MatchDomains) != 1 || settings.PrimaryDNS != "10.0.0.1" || active.ExitNode("my-org") != 9 {
		t.Errorf("settings didn't round trip: %+v", settings)
	}

	active.SetServerSettings("my-org", nil)
	if active.ServerSettings("my-org") != nil {
		t.Error("clearing should remove the settings")
	}
	if active.ExitNode("my-org") != 9 {
		t.Error("clearing server settings must keep the user's exit node")
	}
}
