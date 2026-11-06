package companion

import "runtime"

const (
	windowsMinDesktopAppVersion = "0.11.0"
	darwinMinDesktopAppVersion  = ""
	// Every Pangolin Linux client release supports companion mode.
	linuxMinDesktopAppVersion = ""
)

// RequiredDesktopAppVersion returns the minimum desktop app version required for companion mode on this platform.
func RequiredDesktopAppVersion() string {
	switch runtime.GOOS {
	case "windows":
		return windowsMinDesktopAppVersion
	case "darwin":
		return darwinMinDesktopAppVersion
	case "linux":
		return linuxMinDesktopAppVersion
	default:
		return ""
	}
}
