//go:build !windows

package client

import "errors"

// runWindowsMachineClient is never actually called on non-Windows
// platforms - clientUpMain only invokes it behind a runtime.GOOS ==
// "windows" check - but the symbol must still exist here since client.go
// references it unconditionally (it has no build tag of its own).
func runWindowsMachineClient(opts *ClientUpCmdOpts) error {
	return errors.New("runWindowsMachineClient is only supported on Windows")
}
