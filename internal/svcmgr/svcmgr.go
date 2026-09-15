// Package svcmgr manages a persistent background service that supervises a
// `pangolin up site` or `pangolin up client` process, restarting it
// automatically on crash or reboot. It's backed by systemd on Linux,
// launchd on macOS, and a native Windows Service on Windows - each platform
// file implements the same Install/Uninstall/Status/Follow functions over
// the shared Spec type below.
package svcmgr

// Well-known service names shared between the cmd/service package (which
// builds Specs) and the platform backends (which sometimes need to special-
// case a specific service - see svcmgr_windows.go's client handling).
const (
	SiteServiceName   = "pangolin-site"
	ClientServiceName = "pangolin-client"
)

// Spec describes a service to install.
type Spec struct {
	// Name is a unique, filesystem/registry-safe identifier for the
	// service, e.g. "pangolin-site" or "pangolin-client".
	Name string
	// DisplayName is a human-readable name shown by the OS service manager.
	DisplayName string
	// Description is a short human-readable summary of what the service does.
	Description string
	// Args are the arguments passed to the current pangolin executable to
	// start the supervised process, e.g. []string{"up", "site"} or
	// []string{"up", "client", "--attach"}. They must not contain
	// credentials - use EnvVars for those.
	Args []string
	// EnvVars are set in the supervised process's environment - used to
	// pass credentials without putting them on the command line, where
	// they'd be visible to any local user (e.g. via `ps`).
	EnvVars map[string]string
}
