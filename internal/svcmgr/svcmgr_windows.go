//go:build windows

package svcmgr

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

var ErrElevationRequired = errors.New("this command must be run from an elevated (Administrator) prompt")

// hostMarker and olmHostMarker are private argv[1] sentinels that tell
// main() this process was launched by the Service Control Manager to host
// a service installed by Install below (see MaybeRunHostedService), rather
// than being a normal CLI invocation. They're embedded directly into the
// service's registered BinaryPathName, so they're present on every start -
// including at boot, unlike arguments passed via the SCM's separate
// StartService parameters.
//
// hostMarker supervises a plain `pangolin up ...` child process (used for
// the site service, whose "up site" is fully supported as a standalone
// Windows console command already). olmHostMarker is used only for the
// client service: `pangolin up client` has no standalone Windows console
// mode (see runOlmClient below for why), so that service instead runs the
// olm tunnel directly in-process, the same way olm's own Windows service
// (github.com/fosrl/olm's service_windows.go) does.
const (
	hostMarker    = "__pangolin-service-host"
	olmHostMarker = "__pangolin-service-host-olm"
)

func logDir() string {
	return filepath.Join(os.Getenv("PROGRAMDATA"), "pangolin", "logs")
}

func logPath(name string) string {
	return filepath.Join(logDir(), name+".log")
}

// CheckSupported returns an error if the process isn't running elevated.
func CheckSupported() error {
	if !windows.GetCurrentProcessToken().IsElevated() {
		return ErrElevationRequired
	}
	return nil
}

// Install registers a Windows Service that, on start, re-launches this
// same pangolin executable to host the service - see MaybeRunHostedService.
// For most services that means supervising `spec.Args` (e.g. "up site") as
// a child process; the client service is a special case that instead runs
// the olm tunnel directly in-process (see olmclient_windows.go). Either
// way, credentials are stored in the service's registry Environment value
// rather than on the command line, so they don't show up in Task Manager
// for other users.
func Install(spec Spec) error {
	if err := CheckSupported(); err != nil {
		return err
	}

	exepath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	if s, err := m.OpenService(spec.Name); err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists; run 'pangolin service uninstall' first to reinstall it", spec.Name)
	}

	config := mgr.Config{
		ServiceType:  0x10, // SERVICE_WIN32_OWN_PROCESS
		StartType:    mgr.StartAutomatic,
		ErrorControl: mgr.ErrorNormal,
		DisplayName:  spec.DisplayName,
		Description:  spec.Description,
	}

	// The client service hosts the olm tunnel in-process (see
	// MaybeRunHostedService / runOlmClient) and takes its credentials from
	// the environment only, so no upArgs need embedding for it.
	marker := hostMarker
	if spec.Name == ClientServiceName {
		marker = olmHostMarker
	}
	hostArgs := append([]string{marker, spec.Name}, spec.Args...)

	s, err := m.CreateService(spec.Name, exepath, config, hostArgs...)
	if err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}
	defer s.Close()

	if err := eventlog.InstallAsEventCreate(spec.Name, eventlog.Error|eventlog.Warning|eventlog.Info); err != nil {
		// Non-fatal: the service still works, just without a friendly
		// Event Log source (falls back to a generic one).
		fmt.Printf("Warning: failed to install event log source: %v\n", err)
	}

	if len(spec.EnvVars) > 0 {
		if err := setServiceEnvironment(spec.Name, spec.EnvVars); err != nil {
			s.Delete()
			return fmt.Errorf("failed to set service environment: %w", err)
		}
	}

	if err := os.MkdirAll(logDir(), 0o755); err != nil {
		fmt.Printf("Warning: failed to create log directory: %v\n", err)
	}

	// Best-effort: makes the SCM relaunch the service if it ever exits with
	// a non-zero code (e.g. an unrecoverable auth error, or an unexpected
	// crash) - a safety net on top of each host's own internal restart loop.
	if err := configureRecoveryActions(s); err != nil {
		fmt.Printf("Warning: failed to configure service recovery actions: %v\n", err)
	}

	if err := s.Start(); err != nil {
		return fmt.Errorf("service created but failed to start: %w", err)
	}

	return nil
}

// configureRecoveryActions tells the Service Control Manager to relaunch
// the service automatically when it stops with a non-zero exit code.
func configureRecoveryActions(s *mgr.Service) error {
	actions := []mgr.RecoveryAction{
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 5 * time.Second},
		{Type: mgr.ServiceRestart, Delay: 30 * time.Second},
	}
	if err := s.SetRecoveryActions(actions, 24*60*60); err != nil {
		return fmt.Errorf("failed to set recovery actions: %w", err)
	}
	// By default the SCM only runs recovery actions when a service crashes.
	// Our hosts report a non-crash failure exit code intentionally (see
	// runOlmClient's Execute), so this must be enabled for that to count.
	if err := s.SetRecoveryActionsOnNonCrashFailures(true); err != nil {
		return fmt.Errorf("failed to enable recovery actions on non-crash failures: %w", err)
	}
	return nil
}

// setServiceEnvironment writes the service's registry Environment value
// (HKLM\SYSTEM\CurrentControlSet\Services\<name>\Environment), which the
// SCM merges into the service process's environment at launch.
func setServiceEnvironment(name string, envVars map[string]string) error {
	keyPath := `SYSTEM\CurrentControlSet\Services\` + name
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", k, envVars[k]))
	}

	return key.SetStringsValue("Environment", lines)
}

// Uninstall stops and deletes the service. It does not fail if the service
// was never installed.
func Uninstall(name string) error {
	if err := CheckSupported(); err != nil {
		return err
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()

	if status, err := s.Query(); err == nil && status.State != svc.Stopped {
		if _, err := s.Control(svc.Stop); err == nil {
			timeout := time.Now().Add(30 * time.Second)
			for status.State != svc.Stopped && timeout.After(time.Now()) {
				time.Sleep(300 * time.Millisecond)
				if status, err = s.Query(); err != nil {
					break
				}
			}
		}
	}

	if err := s.Delete(); err != nil {
		return fmt.Errorf("failed to delete service: %w", err)
	}

	_ = eventlog.Remove(name)

	return nil
}

// Status returns a human-readable summary of the service's current state.
func Status(name string) (string, error) {
	if err := CheckSupported(); err != nil {
		return "", err
	}

	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("failed to connect to service manager: %w", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return "Not installed\n", nil
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return "", fmt.Errorf("failed to query service status: %w", err)
	}

	return fmt.Sprintf("State: %s\n", stateString(status.State)), nil
}

func stateString(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "Starting"
	case svc.StopPending:
		return "Stopping"
	case svc.Running:
		return "Running"
	case svc.ContinuePending:
		return "Continue Pending"
	case svc.PausePending:
		return "Pause Pending"
	case svc.Paused:
		return "Paused"
	default:
		return "Unknown"
	}
}

// Follow streams the service's log file to stdout until interrupted.
func Follow(name string, lines int) error {
	return TailFile(logPath(name), lines)
}

// MaybeRunHostedService checks whether this process was launched by the
// Service Control Manager to host a service installed via Install above,
// and if so, runs it (blocking until the service is stopped) and returns
// true. main() should return immediately when this returns true.
func MaybeRunHostedService() bool {
	if len(os.Args) < 3 {
		return false
	}

	name := os.Args[2]

	switch os.Args[1] {
	case hostMarker:
		// Supervises a plain child process running `pangolin up site` -
		// exactly the same command Linux/macOS run under systemd/launchd.
		// This keeps site exercising the same well-tested CLI code path on
		// every platform.
		upArgs := append([]string{}, os.Args[3:]...)
		_ = svc.Run(name, &hostedService{name: name, upArgs: upArgs})
		return true
	case olmHostMarker:
		// Runs the olm tunnel directly in-process - see runOlmClient.
		_ = svc.Run(name, &olmHostedService{name: name})
		return true
	default:
		return false
	}
}

type hostedService struct {
	name   string
	upArgs []string
}

func (h *hostedService) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	elog, elogErr := eventlog.Open(h.name)
	if elogErr == nil {
		defer elog.Close()
		elog.Info(1, fmt.Sprintf("Starting %s (%s)", h.name, strings.Join(h.upArgs, " ")))
	}

	if err := os.MkdirAll(logDir(), 0o755); err != nil && elogErr == nil {
		// nothing to log to; carry on, the child's output is just dropped.
	}
	logFile, _ := os.OpenFile(logPath(h.name), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if logFile != nil {
		defer logFile.Close()
	}

	stopCh := make(chan struct{})
	doneCh := make(chan struct{})
	go h.supervise(logFile, stopCh, doneCh)

	changes <- svc.Status{State: svc.Running, Accepts: accepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				changes <- svc.Status{State: svc.StopPending}
				close(stopCh)
				select {
				case <-doneCh:
				case <-time.After(20 * time.Second):
				}
				if elogErr == nil {
					elog.Info(1, fmt.Sprintf("%s stopped", h.name))
				}
				return false, 0
			}
		case <-doneCh:
			return false, 0
		}
	}
}

// supervise runs the actual `pangolin up ...` child process, restarting it
// (after a short backoff) if it exits unexpectedly, until stopCh is closed.
// This is the Windows equivalent of systemd's Restart=always / launchd's
// KeepAlive.
func (h *hostedService) supervise(logFile *os.File, stopCh <-chan struct{}, doneCh chan<- struct{}) {
	defer close(doneCh)

	exePath, err := os.Executable()
	if err != nil {
		return
	}

	for {
		cmd := exec.Command(exePath, h.upArgs...)
		if logFile != nil {
			cmd.Stdout = logFile
			cmd.Stderr = logFile
		}
		// Its own process group so a CTRL_BREAK_EVENT sent below targets
		// only this child, not the service host itself.
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}

		if err := cmd.Start(); err == nil {
			exitCh := make(chan struct{})
			go func() {
				_ = cmd.Wait()
				close(exitCh)
			}()

			select {
			case <-stopCh:
				// Ask nicely first (equivalent of SIGTERM), then force-kill
				// if it doesn't exit in time.
				_ = windows.GenerateConsoleCtrlEvent(windows.CTRL_BREAK_EVENT, uint32(cmd.Process.Pid))
				select {
				case <-exitCh:
				case <-time.After(10 * time.Second):
					_ = cmd.Process.Kill()
					<-exitCh
				}
				return
			case <-exitCh:
			}
		}

		select {
		case <-stopCh:
			return
		case <-time.After(2 * time.Second):
		}
	}
}
