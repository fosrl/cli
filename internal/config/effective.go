package config

import (
	"strconv"
	"strings"
)

// Defaults `pangolin up` uses for a setting nothing else sets. They're the
// defaults of its flags.
const (
	DefaultTunnelDNS               = false
	DefaultOverrideDNS             = true
	DefaultPreferLocalRoutes       = false
	DefaultExitNodeTakesPrecedence = false
	DefaultMTU                     = 1280
)

// SettingSource says where a setting's effective value comes from.
type SettingSource string

const (
	SourceDefault  SettingSource = "default"
	SourceGlobal   SettingSource = "global"
	SourceOrg      SettingSource = "org"
	SourceServer   SettingSource = "server"
	SourceSelected SettingSource = "selected" // exit node chosen with `pangolin select exit-node`
)

// EffectiveSetting is one setting as each source has it and the value that
// applies. An empty Global, Org or Server means that source doesn't set it.
type EffectiveSetting struct {
	Key       string        `json:"key"`
	Global    string        `json:"global,omitempty"`
	Org       string        `json:"org,omitempty"`
	Server    string        `json:"server,omitempty"`
	Effective string        `json:"effective"`
	Source    SettingSource `json:"source"`
}

// EffectiveSettings resolves every per-org setting for orgID the way
// `pangolin up` does without flags: server settings > the org's entry in the
// config file > the global config > defaults. The exit node is the exception;
// see EffectiveExitNode. selectedExitNode is the exit node saved by
// `pangolin select exit-node`, or 0.
func EffectiveSettings(cfg *Config, orgID string, server *ClientSettings, selectedExitNode int) []EffectiveSetting {
	settings := make([]EffectiveSetting, 0, len(OrgConfigOptions))
	for _, key := range OrgConfigOptions {
		s := EffectiveSetting{Key: key}
		if key != "up.exit_node_resource_id" {
			s.Global, _ = cfg.GetKey(key)
		}
		s.Org, _ = cfg.GetOrgKey(orgID, key)
		s.Server = serverValue(server, key)

		switch {
		case key == "up.exit_node_resource_id":
			s.Effective, s.Source = effectiveExitNode(server, selectedExitNode, cfg.Orgs[orgID])
		case server.ManagesKey(key):
			s.Effective, s.Source = s.Server, SourceServer
		case s.Org != "":
			s.Effective, s.Source = s.Org, SourceOrg
		case s.Global != "":
			s.Effective, s.Source = s.Global, SourceGlobal
		default:
			s.Effective, s.Source = defaultValue(key), SourceDefault
		}
		settings = append(settings, s)
	}
	return settings
}

func effectiveExitNode(server *ClientSettings, selected int, org OrgConfig) (string, SettingSource) {
	id := EffectiveExitNode(server, selected, org)
	var source SettingSource
	switch {
	case id == 0:
		return "", SourceDefault
	case server.LockedExitNode() != 0:
		source = SourceServer
	case selected != 0:
		source = SourceSelected
	case org.ExitNodeResourceID != nil:
		source = SourceOrg
	default:
		source = SourceServer
	}
	return strconv.Itoa(id), source
}

// serverValue formats what the server sets for key, or "" if it doesn't.
func serverValue(s *ClientSettings, key string) string {
	if s == nil {
		return ""
	}
	formatBool := func(b *bool) string {
		if b == nil {
			return ""
		}
		return strconv.FormatBool(*b)
	}

	switch key {
	case "up.tunnel_dns":
		return formatBool(s.TunnelDNS)
	case "up.override_dns":
		return formatBool(s.OverrideDNS)
	case "up.upstream_dns":
		return strings.Join(s.UpstreamDNS(), ",")
	case "up.match_domains_dns":
		return strings.Join(s.MatchDomains, ",")
	case "up.prefer_local_routes":
		return formatBool(s.PreferLocalRoutes)
	case "up.exit_node_takes_precedence":
		return formatBool(s.ExitNodeTakesPrecedence)
	case "up.mtu":
		if s.MTU == nil {
			return ""
		}
		return strconv.Itoa(*s.MTU)
	case "up.exit_node_resource_id":
		// Unlocked, the server's exit node is only a default.
		if s.ExitNodeSiteResourceID == 0 {
			return ""
		}
		if s.ExitNodeLocked {
			return strconv.Itoa(s.ExitNodeSiteResourceID) + " (locked)"
		}
		return strconv.Itoa(s.ExitNodeSiteResourceID) + " (default)"
	default:
		return ""
	}
}

// defaultValue is what `pangolin up` uses when nothing sets key; "" for the
// list settings and the exit node, which default to none.
func defaultValue(key string) string {
	switch key {
	case "up.tunnel_dns":
		return strconv.FormatBool(DefaultTunnelDNS)
	case "up.override_dns":
		return strconv.FormatBool(DefaultOverrideDNS)
	case "up.prefer_local_routes":
		return strconv.FormatBool(DefaultPreferLocalRoutes)
	case "up.exit_node_takes_precedence":
		return strconv.FormatBool(DefaultExitNodeTakesPrecedence)
	case "up.mtu":
		return strconv.Itoa(DefaultMTU)
	default:
		return ""
	}
}
