//go:build !windows && !linux

package companion

func desktopAccountsHint(dataDir string) (activeUserID string, accountCount int) {
	return "", 0
}
