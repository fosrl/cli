//go:build !windows

package companion

func desktopAccountsHint(dataDir string) (activeUserID string, accountCount int) {
	return "", 0
}
