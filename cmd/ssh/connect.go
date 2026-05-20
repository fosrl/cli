package ssh

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fosrl/cli/internal/olm"
)

const (
	siteAppearTimeout  = 15 * time.Second
	siteConnectTimeout = 30 * time.Second
	pollInterval       = 500 * time.Millisecond
)

// siteConnectedMsg is sent to the bubbletea program when any site connects.
type siteConnectedMsg struct{}

// siteConnectTimedOutMsg is sent when the connection poll deadline is exceeded.
type siteConnectTimedOutMsg struct{}

// connectSpinnerModel is a minimal bubbletea model that displays a spinner
// while a background goroutine polls for the site connection.
type connectSpinnerModel struct {
	spinner  spinner.Model
	timedOut bool
}

func newConnectSpinnerModel() connectSpinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	return connectSpinnerModel{spinner: s}
}

func (m connectSpinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m connectSpinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case siteConnectedMsg:
		return m, tea.Quit
	case siteConnectTimedOutMsg:
		m.timedOut = true
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m connectSpinnerModel) View() string {
	return fmt.Sprintf("%s Connecting...\n", m.spinner.View())
}

// waitForAnySiteConnection waits for at least one site from siteIDs to appear
// in the olm status output and become connected.
//
// Phase 1 (up to 10 s): wait for any site ID to appear in PeerStatuses.
// If none appear, the JIT connect call most likely failed server-side.
//
// Phase 2 (up to 30 s): if sites appeared but none are connected yet, show a
// spinner and keep polling until any one connects or the deadline is exceeded.
func waitForAnySiteConnection(client *olm.Client, siteIDs []int) error {
	// ── Phase 1: wait for any site to appear in status ──────────────────────
	deadline := time.Now().Add(siteAppearTimeout)
	appearedIDs := map[int]bool{}
	anyConnected := false

	for time.Now().Before(deadline) {
		status, err := client.GetStatus()
		if err == nil {
			for _, siteID := range siteIDs {
				if peer, ok := status.PeerStatuses[siteID]; ok {
					appearedIDs[siteID] = true
					if peer.Connected {
						anyConnected = true
					}
				}
			}
		}
		if len(appearedIDs) > 0 {
			break
		}
		time.Sleep(pollInterval)
	}

	if len(appearedIDs) == 0 {
		return fmt.Errorf("no sites were added to the connection; the JIT connect request may have failed")
	}

	// At least one site is already connected — nothing more to do.
	if anyConnected {
		return nil
	}

	// ── Phase 2: sites appeared, wait for any to become connected ───────────
	model := newConnectSpinnerModel()
	program := tea.NewProgram(model)

	go func() {
		deadline := time.Now().Add(siteConnectTimeout)
		for time.Now().Before(deadline) {
			status, err := client.GetStatus()
			if err == nil {
				for siteID := range appearedIDs {
					if peer, ok := status.PeerStatuses[siteID]; ok && peer.Connected {
						program.Send(siteConnectedMsg{})
						return
					}
				}
			}
			time.Sleep(pollInterval)
		}
		program.Send(siteConnectTimedOutMsg{})
	}()

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("spinner error: %w", err)
	}

	if finalModel.(connectSpinnerModel).timedOut {
		return fmt.Errorf("Timed out waiting for site to connect. Please disconnect (down) then reconnect (up) the client and try again.")
	}

	return nil
}
