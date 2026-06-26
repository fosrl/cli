//go:build windows

package companion

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type accountsMetadata struct {
	ActiveUserID string `json:"activeUserId"`
	Accounts     map[string]struct {
		UserID string `json:"userId"`
	} `json:"accounts"`
}

func desktopAccountsHint(dataDir string) (activeUserID string, accountCount int) {
	if dataDir == "" {
		return "", 0
	}

	data, err := os.ReadFile(filepath.Join(dataDir, accountsFileName))
	if err != nil {
		return "", 0
	}

	var file accountsMetadata
	if err := json.Unmarshal(data, &file); err != nil {
		return "", 0
	}

	if file.Accounts == nil {
		return file.ActiveUserID, 0
	}
	return file.ActiveUserID, len(file.Accounts)
}
