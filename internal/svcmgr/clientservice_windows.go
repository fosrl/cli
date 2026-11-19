//go:build windows

package svcmgr

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	newtLogger "github.com/fosrl/newt/logger"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

// inProcessService is the Windows Service handler for the client service.
// It runs `pangolin up client --attach` (upArgs) through the normal CLI, the
// same command systemd/launchd run on Linux/macOS, but inside the service
// process rather than as a child: olm refuses to create its TUN interface
// unless the process itself is a Windows service (see newt's
// clients/permissions), so a supervised child like the site service's
// can't work here.
type inProcessService struct {
	name   string
	upArgs []string
}

func (h *inProcessService) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	elog, elogErr := eventlog.Open(h.name)
	if elogErr == nil {
		defer elog.Close()
		elog.Info(1, fmt.Sprintf("Starting %s (%v)", h.name, h.upArgs))
	}

	// A service has no console: send everything the CLI and olm print to
	// the log file `pangolin service logs` follows, as the site service's
	// supervisor does with its child's output.
	if err := os.MkdirAll(logDir(), 0o755); err == nil {
		if logFile, err := os.OpenFile(logPath(h.name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			defer logFile.Close()
			redirectOutput(logFile)
		}
	}

	if RunCLI == nil {
		if elogErr == nil {
			elog.Error(1, "RunCLI is not set; cannot run the client")
		}
		changes <- svc.Status{State: svc.StopPending}
		return false, 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	doneCh := make(chan error, 1)
	go func() { doneCh <- RunCLI(ctx, h.upArgs) }()

	changes <- svc.Status{State: svc.Running, Accepts: accepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				cancel()
				select {
				case <-doneCh:
				case <-time.After(15 * time.Second):
				}
				if elogErr == nil {
					elog.Info(1, fmt.Sprintf("%s stopped", h.name))
				}
				return false, 0
			}
		case err := <-doneCh:
			// The client ended on its own rather than through a Stop
			// request. Report a failure so the recovery actions set in
			// Install relaunch the service, matching systemd's
			// Restart=always / launchd's KeepAlive. (Paths in the client
			// that call os.Exit end the process instead; the SCM treats
			// that as a crash and relaunches it the same way.)
			if err == nil {
				err = errors.New("exited")
			}
			if elogErr == nil {
				elog.Error(1, fmt.Sprintf("%s: client stopped unexpectedly: %v", h.name, err))
			}
			changes <- svc.Status{State: svc.StopPending}
			return false, 1
		}
	}
}

// redirectOutput points this process's stdout and stderr, both the Go
// variables and the Windows standard handles, plus olm's logger at f.
func redirectOutput(f *os.File) {
	os.Stdout = f
	os.Stderr = f
	_ = windows.SetStdHandle(windows.STD_OUTPUT_HANDLE, windows.Handle(f.Fd()))
	_ = windows.SetStdHandle(windows.STD_ERROR_HANDLE, windows.Handle(f.Fd()))
	newtLogger.GetLogger().SetOutput(f)
}
