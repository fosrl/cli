//go:build windows

package svcmgr

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	versionpkg "github.com/fosrl/cli/internal/version"
	newtLogger "github.com/fosrl/newt/logger"
	olmpkg "github.com/fosrl/olm/olm"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

// olmHostedService is the Windows Service handler for the client (Olm)
// service. Unlike hostedService (used for site), it doesn't spawn a
// `pangolin up client` child process - that command has no standalone
// Windows console mode, since interactive tunnel management on Windows is
// the Pangolin desktop app's job, not the CLI's. Instead, mirroring how
// olm's own Windows service (github.com/fosrl/olm's service_windows.go)
// works, it runs the tunnel directly in-process via the vendored olm
// package.
type olmHostedService struct {
	name string
}

func (h *olmHostedService) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	elog, elogErr := eventlog.Open(h.name)
	if elogErr == nil {
		defer elog.Close()
		elog.Info(1, "Starting Pangolin client tunnel")
	}

	creds := ClientCreds{
		ID:       os.Getenv("PANGOLIN_CLIENT_ID"),
		Secret:   os.Getenv("PANGOLIN_CLIENT_SECRET"),
		Endpoint: os.Getenv("PANGOLIN_ENDPOINT"),
		OrgID:    os.Getenv("PANGOLIN_ORG"),
	}
	if creds.ID == "" || creds.Secret == "" || creds.Endpoint == "" {
		if elogErr == nil {
			elog.Error(1, "Missing PANGOLIN_CLIENT_ID/PANGOLIN_CLIENT_SECRET/PANGOLIN_ENDPOINT in the service environment")
		}
		changes <- svc.Status{State: svc.StopPending}
		return false, 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan error, 1)
	go func() { doneCh <- runOlmClient(ctx, h.name, creds) }()

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
					elog.Info(1, "Pangolin client tunnel stopped")
				}
				return false, 0
			}
		case err := <-doneCh:
			// The tunnel ended on its own (auth error, fatal error, or an
			// API-initiated exit) rather than via a Stop/Shutdown control
			// request. Report a non-zero exit so the recovery actions
			// configured in Install relaunch the service, matching
			// systemd's Restart=always / launchd's KeepAlive.
			cancel()
			if elogErr == nil {
				elog.Error(1, fmt.Sprintf("Pangolin client tunnel exited unexpectedly: %v", err))
			}
			changes <- svc.Status{State: svc.StopPending}
			return false, 1
		}
	}
}

// ClientCreds are the machine-client credentials the olm-hosted Windows
// service needs. Unlike `pangolin up client`'s full flow (account login,
// keyring, interactive detach/TUI), this only ever runs in machine-client
// mode - the same restriction `pangolin service install site` already
// places on the site service (explicit --id/--secret/--endpoint, no
// account needed).
type ClientCreds struct {
	ID       string
	Secret   string
	Endpoint string
	OrgID    string
}

// runOlmClient runs the olm tunnel in-process until ctx is cancelled or the
// tunnel ends on its own. It mirrors the core of olm's own console-mode
// startup (see olm/main.go's runOlmMainWithArgs / the tail of
// clientUpMain's attached-mode path in cmd/up/client), trimmed to the
// machine-client case only - no config file, no keyring, no fingerprinting
// (those only apply to interactively logged-in user devices).
func runOlmClient(ctx context.Context, serviceName string, creds ClientCreds) error {
	newtLogger.Init(nil)
	if logFile, err := os.OpenFile(logPath(serviceName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
		defer logFile.Close()
		newtLogger.GetLogger().SetOutput(logFile)
	}

	// finish reports the tunnel's terminal outcome exactly once, whichever
	// of olm's callbacks (or ctx cancellation) fires first.
	var once sync.Once
	doneCh := make(chan error, 1)
	finish := func(err error) {
		once.Do(func() { doneCh <- err })
	}

	olmInitConfig := olmpkg.OlmConfig{
		LogLevel:   "info",
		EnableAPI:  true,
		// Named pipe on Windows. A name distinct from olm's own "olm"
		// default, so this machine-client service can't collide with a
		// separate interactive olm/desktop-app instance on the same box.
		SocketPath: "pangolin-client",
		Version:    versionpkg.Version,
		Agent:      "Pangolin Service (Olm)",
		// No WatchdogSubcommand: `pangolin up client` sets one (see
		// cmd/watchdog) as a DNS-cleanup safety net for its own process,
		// but the CLI's watchdog subcommand isn't implemented on Windows
		// (cmd/watchdog/watchdog_windows.go is a stub), so pointing olm at
		// it here would just fail to spawn. A clean Stop still restores DNS
		// via o.Close() below; only an unclean crash of this process would
		// leave the override in place - an acceptable gap for now.
		OnTerminated: func() { finish(nil) },
		OnAuthError: func(statusCode int, message string) {
			finish(fmt.Errorf("authentication error: %d %s", statusCode, message))
		},
		OnExit: func() { finish(nil) },
	}

	o, err := olmpkg.Init(ctx, olmInitConfig)
	if err != nil {
		return fmt.Errorf("failed to init olm: %w", err)
	}
	defer o.Close()

	if err := o.StartApi(); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	tunnelConfig := olmpkg.TunnelConfig{
		Endpoint:             creds.Endpoint,
		ID:                   creds.ID,
		Secret:               creds.Secret,
		OrgID:                creds.OrgID,
		MTU:                  1280,
		InterfaceName:        "pangolin",
		Holepunch:            true,
		PingIntervalDuration: 5 * time.Second,
		PingTimeoutDuration:  5 * time.Second,
		OverrideDNS:          true,
		EnableUAPI:           false,
	}
	go o.StartTunnel(tunnelConfig)

	select {
	case <-ctx.Done():
		return nil
	case err := <-doneCh:
		return err
	}
}
