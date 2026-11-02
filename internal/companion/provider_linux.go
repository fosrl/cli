//go:build linux

package companion

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

// The Pangolin Linux client keeps its per-user files under
// $XDG_CONFIG_HOME/pangolin-client (normally ~/.config/pangolin-client), apart
// from the CLI's ~/.config/pangolin.
const linuxAppDirName = "pangolin-client"

// linuxAppBinary is the desktop client's executable, installed by its
// deb, rpm and Arch packages.
const linuxAppBinary = "pangolin-client"

func defaultDesktopDataDir() string {
	if u := sudoUser(); u != nil {
		return filepath.Join(u.HomeDir, ".config", linuxAppDirName)
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, linuxAppDirName)
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", linuxAppDirName)
	}
	return ""
}

// platformProvider offers companion mode only when the desktop client is
// installed. Most Linux CLI installs have no desktop client (servers, other
// desktops), and turning companion mode on there would lock them out of
// their own CLI logins.
func platformProvider() Provider {
	if !desktopAppInstalled() {
		return nil
	}
	return desktopProvider{secrets: newSocketSecretsClient()}
}

func desktopAppInstalled() bool {
	if _, err := os.Stat(linuxDaemonSocketPath); err == nil {
		return true
	}
	_, err := exec.LookPath(linuxAppBinary)
	return err == nil
}

// sudoUser returns the user who ran sudo, when this process is root under
// sudo, so the CLI uses that user's desktop session rather than root's.
func sudoUser() *user.User {
	if os.Geteuid() != 0 {
		return nil
	}
	uid := os.Getenv("SUDO_UID")
	if uid == "" {
		return nil
	}
	u, err := user.LookupId(uid)
	if err != nil {
		return nil
	}
	return u
}
