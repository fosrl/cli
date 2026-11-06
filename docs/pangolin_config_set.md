## pangolin config set

Set a config value

### Synopsis

Set a config value and write it to the config file.

A value can apply to every organization (--global) or to one (--org), which
takes precedence over the global value. Without either flag you're asked when
running interactively with an organization selected; otherwise it's global.
Settings managed by your organization's server take precedence over both.

Supported keys:
  log_level
  disable_update_check
  disable_companion_mode
  up.tunnel_dns
  up.upstream_dns
  up.override_dns
  up.match_domains_dns
  up.prefer_local_routes
  up.exit_node_takes_precedence
  up.mtu
  session_cookie_name

Per-organization only:
  up.exit_node_resource_id

Examples:
  pangolin config set up.tunnel_dns true
  pangolin config set up.upstream_dns 10.0.0.53
  pangolin config set up.upstream_dns 10.0.0.53,10.0.0.54
  pangolin config set up.mtu 1400 --org my-org


```
pangolin config set <key> <value> [flags]
```

### Options

```
      --global       Set the value for all organizations
  -h, --help         help for set
      --org string   Set the value for this organization only
```

### SEE ALSO

* [pangolin config](pangolin_config.md)	 - View and edit CLI configuration

