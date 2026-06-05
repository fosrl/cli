## pangolin scp

Run scp using just-in-time SSH certificates

### Synopsis

Run scp(1) in the terminal. Generates a key pair and signs it just-in-time, then executes the system OpenSSH scp client.

Use the resource alias or identifier as the host in remote operands, exactly as you would with regular scp.
The resource alias is resolved to the connected hostname transparently.
Examples:
  pangolin scp ./local-file my-server.internal:/remote/path
  pangolin scp my-server.internal:/var/log/syslog ./syslog
  pangolin scp -r ./dir my-server.internal:~/

Set PANGOLIN_SCP_BINARY to the full path of scp(1) to override PATH lookup on all platforms.

```
pangolin scp [flags] <source> <destination>
```

### Options

```
  -h, --help       help for scp
  -p, --port int   Remote SCP/SSH port (default: 22)
```

### SEE ALSO

* [pangolin](pangolin.md)	 - Pangolin CLI

