## pangolin service install client

Install and start the client background service

### Synopsis

Install a background service for this machine client, then start it
immediately.

Intended for machine clients (--id/--secret), which don't have an
interactively logged-in user to restart them. On Windows this only supports
the service itself - 'pangolin up client' has no standalone Windows console
mode (that's handled by the Pangolin desktop app); the service runs the
tunnel directly in-process instead.

```
pangolin service install client [flags]
```

### Options

```
      --endpoint string   Pangolin server endpoint
  -h, --help              help for client
      --id string         Client ID
      --org string        Organization ID
      --secret string     Client secret
```

### SEE ALSO

* [pangolin service install](pangolin_service_install.md)	 - Install and start a background service

