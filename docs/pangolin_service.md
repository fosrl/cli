## pangolin service

Manage background services that keep a site or client running persistently

### Synopsis

Install, remove, and monitor background services that run 'pangolin up site'
or 'pangolin up client' persistently, restarting them automatically if they
crash or the machine reboots.

Backed by systemd on Linux, launchd on macOS, and a Windows Service on
Windows. Must be run as root (Linux/macOS) or from an elevated prompt
(Windows). The client (machine client) service isn't available on Windows
yet, since 'pangolin up client' itself doesn't support Windows.

### Options

```
  -h, --help   help for service
```

### SEE ALSO

* [pangolin](pangolin.md)	 - Pangolin CLI
* [pangolin service install](pangolin_service_install.md)	 - Install and start a background service
* [pangolin service logs](pangolin_service_logs.md)	 - Follow a background service's logs
* [pangolin service status](pangolin_service_status.md)	 - Show a background service's status
* [pangolin service uninstall](pangolin_service_uninstall.md)	 - Stop and remove a background service

