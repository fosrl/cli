package notice

import (
	"runtime"

	"github.com/fosrl/cli/internal/companion"
	"github.com/fosrl/cli/internal/config"
)

const (
	companionModeIntroID      = "companion-mode-intro-windows-v1"
	companionModeIntroLinuxID = "companion-mode-intro-linux-v1"
	desktopClientLinuxID      = "desktop-client-available-linux-v1"
)

const downloadsURL = "https://pangolin.net/downloads"

var registeredNotices = []Notice{
	{
		ID:         companionModeIntroID,
		MinVersion: "0.11.0",
		MaxVersion: "0.11.0",
		Condition: func(cfg *config.Config) bool {
			return runtime.GOOS == "windows" && companion.PlatformAvailable()
		},
		Lines: companionModeIntroLines,
	},
	{
		// Not tied to a CLI version: Linux offers companion mode only once
		// the desktop client is installed, which can happen at any time.
		ID: companionModeIntroLinuxID,
		Condition: func(cfg *config.Config) bool {
			return runtime.GOOS == "linux" && companion.PlatformAvailable()
		},
		Lines: companionModeIntroLines,
	},
	{
		// Shown once to Linux users without the desktop client. Those who
		// already have it get the companion mode notice above instead.
		ID: desktopClientLinuxID,
		Condition: func(cfg *config.Config) bool {
			return runtime.GOOS == "linux" && !companion.PlatformAvailable()
		},
		Lines: desktopClientLinuxLines,
	},
}

func desktopClientLinuxLines(cfg *config.Config) []string {
	return []string{
		"A Pangolin desktop client is now available for Linux.",
		"It adds a tray icon for connecting, and the CLI can use its login.",
		"Download it at " + downloadsURL,
	}
}

func companionModeIntroLines(cfg *config.Config) []string {
	clientName := companion.PlatformProviderName()

	lines := []string{
		"Pangolin CLI now uses companion mode with " + clientName + ".",
		"Login, logout, and account, organization, or exit node changes are managed in " + clientName + ".",
	}
	if companion.RequiredDesktopAppVersion() != "" {
		lines = append(lines, "Requires "+clientName+" version "+companion.RequiredDesktopAppVersion()+" or later.")
	}
	lines = append(lines, "Run 'pangolin companion status' for details.")
	if cfg.CompanionModeEnabled() {
		lines = append(lines, "To use standalone CLI authentication, run 'pangolin companion disable'.")
	}
	return lines
}
