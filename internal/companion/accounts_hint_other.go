//go:build !windows && !linux && !darwin

package companion

func desktopAccountsHint(dataDir string) (activeUserID string, accountCount int) {
	return "", 0
}
