## pangolin service install site

Install and start the site (Newt) background service

### Synopsis

Install a background service for this site, then start it immediately.

Credentials can be given directly (--id, --secret, --endpoint) or via a
newt config file (--config-file), in which case the flags are optional and
any that are set override the file's values.

```
pangolin service install site [flags]
```

### Options

```
      --config-file string   Path to a newt config file passed through to 'pangolin up site'
      --disable-clients      Disable accepting client connections
      --disable-ssh          Disable Pangolin SSH
      --endpoint string      Pangolin server endpoint (required unless --config-file is set)
  -h, --help                 help for site
      --id string            Site ID (required unless --config-file is set)
      --secret string        Site secret (required unless --config-file is set)
```

### SEE ALSO

* [pangolin service install](pangolin_service_install.md)	 - Install and start a background service

