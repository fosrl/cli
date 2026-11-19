//go:build windows

package client

import (
	"errors"
	"fmt"

	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/svcmgr"
	"golang.org/x/sys/windows/svc"
)

// runWindowsMachineClient is `pangolin up client` without --attach on
// Windows. Instead of a detached subprocess it installs the client service,
// the same one `pangolin service install client` installs, tails its logs
// so the command still feels like it runs the tunnel, and removes the
// service again on Ctrl+C. The service runs `pangolin up client --attach`
// with upArgs, the settings this run resolved; the credentials go in the
// service environment so they don't show up in Task Manager.
func runWindowsMachineClient(id, secret string, upArgs []string) error {
	envVars := map[string]string{
		"PANGOLIN_CLIENT_ID":     id,
		"PANGOLIN_CLIENT_SECRET": secret,
	}

	spec := svcmgr.Spec{
		Name:        svcmgr.ClientServiceName,
		DisplayName: "Pangolin Client",
		Description: "Runs 'pangolin up client' persistently in the background",
		Args:        append([]string{"up", "client", "--attach"}, upArgs...),
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

// requireWindowsService fails unless this process is a Windows service:
// olm only creates its tunnel interface inside one (see newt's
// clients/permissions). The client service runs `up client --attach`
// in-process, so this only fails for --attach run from a console.
func requireWindowsService() error {
	isService, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("could not check whether this is a Windows service: %w", err)
	}
	if !isService {
		return errors.New("on Windows, --attach only works under the client service; run without --attach, or use 'pangolin service install client'")
	}
	return nil
}
