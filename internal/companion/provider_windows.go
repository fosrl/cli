//go:build windows

package companion

import (
	"os"
	"path/filepath"
)

const windowsAppName = "Pangolin"

func platformProvider() Provider {
	appData := os.Getenv("LOCALAPPDATA")
	if appData == "" {
		appData = os.Getenv("APPDATA")
	}
	return desktopProvider{
		defaultDataDir: filepath.Join(appData, windowsAppName),
		secrets: func(string) SecretsClient {
			return newPipeSecretsClient()
		},
	}
}
