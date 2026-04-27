## pangolin ssh sign

Generate and sign an SSH key, then save to files for use with system SSH.

### Synopsis

Generates a key pair, signs the public key, and writes the private key and certificate to files.

```
pangolin ssh sign <resource-id> [flags]
```

### Options

```
      --cert-file string   Path to write the certificate (default: <key-file>-cert.pub)
  -h, --help               help for sign
      --key-file string    Path to write the private key (required)
```

### SEE ALSO

* [pangolin ssh](pangolin_ssh.md)	 - Run an interactive SSH session

