//go:build !windows

package svcmgr

// MaybeRunHostedService always returns false on non-Windows platforms:
// Linux and macOS services directly supervise a plain `pangolin up ...`
// process via systemd/launchd, with no special hosting mode needed (see
// svcmgr_windows.go for why Windows needs one).
func MaybeRunHostedService() bool {
	return false
}
