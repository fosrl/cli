package config

import "testing"

func boolPtr(b bool) *bool { return &b }

func TestEffectiveSettings(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SUDO_USER", "")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}

	for key, value := range map[string]string{
		"up.mtu":          "1400",
		"up.tunnel_dns":   "true",
		"up.upstream_dns": "1.1.1.1",
	} {
		if err := cfg.SetKey(key, value); err != nil {
			t.Fatal(err)
		}
	}
	for key, value := range map[string]string{
		"up.mtu":                   "1380",
		"up.tunnel_dns":            "false",
		"up.exit_node_resource_id": "30",
	} {
		if err := cfg.SetOrgKey("org1", key, value); err != nil {
			t.Fatal(err)
		}
	}
	server := &ClientSettings{
		TunnelDNS:              boolPtr(true),
		PrimaryDNS:             "10.0.0.53",
		ExitNodeSiteResourceID: 10,
	}

	got := map[string]EffectiveSetting{}
	for _, s := range EffectiveSettings(cfg, "org1", server, 20) {
		got[s.Key] = s
	}

	tests := []struct {
		key       string
		effective string
		source    SettingSource
	}{
		{"up.mtu", "1380", SourceOrg},
		{"up.tunnel_dns", "true", SourceServer},
		{"up.upstream_dns", "10.0.0.53", SourceServer},
		{"up.override_dns", "true", SourceDefault},
		{"up.match_domains_dns", "", SourceDefault},
		{"up.exit_node_resource_id", "20", SourceSelected},
	}
	for _, tt := range tests {
		s := got[tt.key]
		if s.Effective != tt.effective || s.Source != tt.source {
			t.Errorf("%s = %q from %s, want %q from %s", tt.key, s.Effective, s.Source, tt.effective, tt.source)
		}
	}

	if s := got["up.mtu"]; s.Global != "1400" || s.Org != "1380" || s.Server != "" {
		t.Errorf("up.mtu sources = %+v", s)
	}
	if s := got["up.exit_node_resource_id"]; s.Server != "10 (default)" || s.Global != "" {
		t.Errorf("exit node sources = %+v", s)
	}

	// An org with no overrides follows the global config.
	for _, s := range EffectiveSettings(cfg, "org2", nil, 0) {
		if s.Key == "up.mtu" && (s.Effective != "1400" || s.Source != SourceGlobal) {
			t.Errorf("org2 up.mtu = %q from %s, want 1400 from global", s.Effective, s.Source)
		}
		if s.Key == "up.exit_node_resource_id" && s.Effective != "" {
			t.Errorf("org2 exit node = %q, want none", s.Effective)
		}
	}
}
