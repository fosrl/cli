//go:build windows

package client

import (
	"fmt"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/svcmgr"
)

// runWindowsMachineClient makes `pangolin up client` work on Windows by
// installing the same background service `pangolin service install client`
// would, tailing its logs so the command still feels like it's running the
// tunnel directly, and uninstalling the service again when interrupted
// (Ctrl+C).
//
// This indirection exists because the olm tunnel can't run directly from a
// plain console process on Windows the way it can on Linux/macOS - it needs
// to run as a Windows Service (see olmHostedService in internal/svcmgr).
// PreRunE already guarantees opts has --id/--secret/--endpoint set before
// this is called (Windows only supports machine clients).
func runWindowsMachineClient(opts *ClientUpCmdOpts) error {
	envVars := map[string]string{
		"PANGOLIN_CLIENT_ID":     opts.ID,
		"PANGOLIN_CLIENT_SECRET": opts.Secret,
		"PANGOLIN_ENDPOINT":      opts.Endpoint,
	}
	if opts.OrgID != "" {
		envVars["PANGOLIN_ORG"] = opts.OrgID
	}

	spec := svcmgr.Spec{
		Name:        svcmgr.ClientServiceName,
		DisplayName: "Pangolin Client",
		Description: "Runs 'pangolin up client' persistently in the background",
		Args:        []string{"up", "client", "--attach"},
		EnvVars:     envVars,
	}

	if err := svcmgr.Install(spec); err != nil {
		return fmt.Errorf("failed to install client service: %w", err)
	}

	logger.Success("Installed and started the %s service", svcmgr.ClientServiceName)
	logger.Info("Press Ctrl+C to stop and remove the service.")
	logger.Info("Use the pangolin service install client command to make it run permanently.")

	// Follow blocks until interrupted (Ctrl+C/SIGTERM, handled internally
	// by TailFile) or the log file otherwise disappears out from under it -
	// either way, once it returns we tear the service back down.
	if err := svcmgr.Follow(svcmgr.ClientServiceName, 20); err != nil {
		logger.Warning("Log following stopped: %v", err)
	}

	logger.Info("Stopping and removing the %s service...", svcmgr.ClientServiceName)
	if err := svcmgr.Uninstall(svcmgr.ClientServiceName); err != nil {
		return fmt.Errorf("failed to remove client service: %w", err)
	}
	logger.Success("Removed the %s service", svcmgr.ClientServiceName)

	return nil
}
