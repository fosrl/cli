package main

import (
	"github.com/fosrl/cli/cmd"
	"github.com/fosrl/cli/internal/svcmgr"
)

func main() {
	// On Windows, a process started by the Service Control Manager to host
	// a service installed via `pangolin service install` never reaches
	// normal CLI dispatch - it runs the service directly and blocks until
	// stopped. No-op on other platforms (see svcmgr_notwindows.go).
	if svcmgr.MaybeRunHostedService() {
		return
	}

	cmd.Execute()
}
