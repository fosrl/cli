## pangolin apply blueprint

Apply a blueprint

### Synopsis

Apply a YAML blueprint to the Pangolin server. --file may be a glob pattern (e.g. -f 'inference-*.yaml') or given multiple times; every matching file is applied one by one.

```
pangolin apply blueprint [flags]
```

### Options

```
      --api-key string     Integration API key (id.secret)
      --endpoint string    Integration API host URL
  -f, --file stringArray   Blueprint YAML file path, or glob pattern (e.g. 'inference-*.yaml'); repeatable. Use '-' for stdin
  -h, --help               help for blueprint
  -n, --name string        Blueprint name (default: filename without extension); only valid for a single file
      --org string         Organization ID
```

### SEE ALSO

* [pangolin apply](pangolin_apply.md)	 - Apply commands

