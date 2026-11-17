//go:build !windows

package client

import "errors"

// runWindowsMachineClient is only called on Windows (behind a runtime.GOOS
// check); client.go references it unconditionally, so the symbol must exist.
func runWindowsMachineClient(id, secret string, upArgs []string) error {
	return errors.New("runWindowsMachineClient is only supported on Windows")
}

// requireWindowsService is only called on Windows; elsewhere the attached
// path checks for root itself.
func requireWindowsService() error {
	return nil
}
