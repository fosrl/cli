//go:build !windows && !linux

package olm

// desktopAppSocketPath is empty where the desktop app's OLM isn't supported.
const desktopAppSocketPath = ""
