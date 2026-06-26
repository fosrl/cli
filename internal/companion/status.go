package companion

import (
	"fmt"

	"github.com/fosrl/cli/internal/config"
)

const (
	statusDisabled    = "disabled"
	statusEnabled     = "enabled"
	statusUnavailable = "unavailable"
)

// StatusReport describes companion mode configuration and desktop client readiness.
type StatusReport struct {
	ConfigState string
	ClientName  string
	Ready       bool
	Suggestion  string
}

// EvaluateStatus probes companion config and desktop client readiness.
func EvaluateStatus(cfg *config.Config) StatusReport {
	if cfg.DisableCompanionMode {
		return StatusReport{ConfigState: statusDisabled}
	}

	provider := platformProvider()
	if provider == nil {
		return StatusReport{
			ConfigState: statusUnavailable,
			ClientName:  "not supported on this platform",
		}
	}

	report := StatusReport{
		ConfigState: statusEnabled,
		ClientName:  provider.Name(),
	}

	dataDir := ResolveDataDir(cfg, provider)
	session, err := provider.LoadSession(dataDir)
	if err != nil {
		report.Suggestion = fmt.Sprintf("Open %s to log in.", provider.Name())
		return report
	}

	if session != nil && session.Active() {
		report.Ready = true
		return report
	}

	report.Suggestion = notReadySuggestion(provider, dataDir, session)
	return report
}
