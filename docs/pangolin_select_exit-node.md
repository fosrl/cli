## pangolin select exit-node

Route all traffic through an exit node

### Synopsis

List the exit nodes in your organization and select one to route all
tunnel traffic (full tunnel) through the sites backing it.

While an exit node is active, a "None" option is shown to turn it off.

With a running client the change takes effect immediately. Without one, the
choice is saved and applied the next time you run 'pangolin up'.

```
pangolin select exit-node [flags]
```

### Options

```
      --exit-node NICE-ID   Exit node NICE-ID to select
  -h, --help                help for exit-node
```

### SEE ALSO

* [pangolin select](pangolin_select.md)	 - Select account information to use

