## pangolin reset-dns

Force-clear stale DNS overrides

### Synopsis

Forcibly clear stale DNS overrides left behind by a crashed or
stuck client. This restores your system DNS to its original
configuration.

By default this command refuses to run when a client is still
active; use --force to override that check.

```
pangolin reset-dns [flags]
```

### Options

```
      --force              Run the reset even if a client appears to be active
  -h, --help               help for reset-dns
      --interface string   Tunnel interface name to clean up (default "pangolin")
```

### SEE ALSO

* [pangolin](pangolin.md)	 - Pangolin CLI

