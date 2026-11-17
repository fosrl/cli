//go:build darwin

package companion

import (
	"path/filepath"

	"github.com/fosrl/cli/internal/config"
)

const darwinAppName = "Pangolin"

func platformProvider() Provider {
	// The original user's home, so the CLI under sudo still finds their app.
	dataDir := ""
	if home, err := config.UserHomeDir(); err == nil {
		dataDir = filepath.Join(home, "Library", "Application Support", darwinAppName)
	}
	return desktopProvider{
		defaultDataDir: dataDir,
		secrets:        newSocketSecretsClient,
	}
}
