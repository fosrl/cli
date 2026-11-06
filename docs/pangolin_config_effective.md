## pangolin config effective

Show the settings that apply to an organization and where each comes from

### Synopsis

Show, for each per-organization setting, the global value, the organization's
own value, the value its server manages, and the value that applies when
connecting with `pangolin up`.

The value that applies is the first set of: the server's, the organization's,
the global one, the default. The exit node is the exception: one the server
locks wins, then the one picked with `pangolin select exit-node`, then the
organization's, then the server's default.

Flags passed to `pangolin up` override global and organization values,
but not ones the server manages.

```
pangolin config effective [flags]
```

### Options

```
  -h, --help         help for effective
      --json         Print as JSON
      --org string   Organization to show (default: selected organization)
```

### SEE ALSO

* [pangolin config](pangolin_config.md)	 - View and edit CLI configuration

