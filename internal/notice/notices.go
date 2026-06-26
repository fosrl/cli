package notice

import (
	"runtime"

	"github.com/fosrl/cli/internal/companion"
	"github.com/fosrl/cli/internal/config"
)

const companionModeIntroID = "companion-mode-intro-windows-v1"

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
}

func companionModeIntroLines(cfg *config.Config) []string {
	clientName := companion.PlatformProviderName()

	lines := []string{
		"Pangolin CLI now uses companion mode with " + clientName + ".",
		"Login, logout, and account or organization changes are managed in " + clientName + ".",
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
