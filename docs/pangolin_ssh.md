## pangolin ssh

Run an interactive SSH session

### Synopsis

Run an SSH client in the terminal. Generates a key pair and signs it just-in-time, then connects to the target resource.

By default the system OpenSSH client is used on every platform. You can pass the same options as ssh(1) after the resource name (for example port forwards: -L, -R, -D, and -N), then an optional remote command. Example: pangolin ssh <resource> -L 8080:127.0.0.1:80 -N

Set PANGOLIN_SSH_BINARY to the full path of ssh(1) to override PATH lookup on all platforms.

```
pangolin ssh <resource alias or identifier> [flags]
```

### Options

```
      --builtin    Use the built-in SSH client instead of the system OpenSSH binary (interactive shell only)
  -h, --help       help for ssh
  -p, --port int   Remote SSH port (default: 22)
```

### SEE ALSO

* [pangolin](pangolin.md)	 - Pangolin CLI
* [pangolin ssh sign](pangolin_ssh_sign.md)	 - Generate and sign an SSH key, then save to files for use with system SSH.

