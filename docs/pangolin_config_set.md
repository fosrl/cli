## pangolin config set

Set a config value

### Synopsis

Set a config value and write it to the config file.

Supported keys:
  log_level
  disable_update_check
  disable_companion_mode
  up.tunnel_dns
  up.upstream_dns
  up.override_dns

Examples:
  pangolin config set up.tunnel_dns true
  pangolin config set up.upstream_dns 10.0.0.53
  pangolin config set up.upstream_dns 10.0.0.53,10.0.0.54


```
pangolin config set <key> <value> [flags]
```

### Options

```
  -h, --help   help for set
```

### SEE ALSO

* [pangolin config](pangolin_config.md)	 - View and edit CLI configuration

