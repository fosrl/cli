package companion

import "runtime"

const (
	windowsMinDesktopAppVersion = "0.11.0"
	darwinMinDesktopAppVersion  = ""
)

// RequiredDesktopAppVersion returns the minimum desktop app version required for companion mode on this platform.
func RequiredDesktopAppVersion() string {
	switch runtime.GOOS {
	case "windows":
		return windowsMinDesktopAppVersion
	case "darwin":
		return darwinMinDesktopAppVersion
	default:
		return ""
	}
}
