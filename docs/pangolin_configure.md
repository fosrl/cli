## pangolin configure

Configure a local AI client to use a Pangolin AI gateway resource

### Synopsis

Writes local config files (e.g. ~/.claude/settings.json) so an AI client talks to a Pangolin
resource acting as an AI gateway. Supported clients: claude, codex, opencode.

If [key] is omitted, an API key is fetched automatically for public resources (private/site
resources need no key). Pass [key] to configure with a credential obtained elsewhere without
making any API calls for it.

```
pangolin configure <client> [key] [flags]
```

### Options

```
  -h, --help              help for configure
      --org string        Organization ID (defaults to your active org)
      --resource string   Resource niceId or domain to configure against (defaults to auto-pick/prompt)
```

### SEE ALSO

* [pangolin](pangolin.md)	 - Pangolin CLI

