package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
)

// ExitCondition determines if the loading indicator should exit.
// Returns whether to exit, and a completion status (true = success).
type ExitCondition func(client *olm.Client, status *olm.StatusResponse) (shouldExit bool, completed bool)

// StatusFormatter formats the status text shown beside the spinner.
type StatusFormatter func(isRunning bool, status *olm.StatusResponse) string

// LoadingConfig configures the loading indicator TUI.
type LoadingConfig struct {
	Header          string
	ExitCondition   ExitCondition
	OnEarlyExit     func(client *olm.Client)                         // Called when user exits early (Ctrl+C)
	OnError         func(client *olm.Client, error *olm.StatusError) // Called when error is detected before registration
	StatusFormatter StatusFormatter                                  // Status formatter (required)
}

var loadingSpinner = spinner.Spinner{
	Frames: []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
	FPS:    time.Second / 10,
}

type loadingModel struct {
	config    LoadingConfig
	olmClient *olm.Client
	status    *olm.StatusResponse
	completed bool
	error     *olm.StatusError
	spinner   spinner.Model
	quitting  bool
}

func NewLoading(config LoadingConfig) (completed bool, statusError *olm.StatusError, err error) {
	s := spinner.New()
	s.Spinner = loadingSpinner
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	model := &loadingModel{
		config:    config,
		olmClient: olm.NewClient(""),
		spinner:   s,
	}

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return false, nil, err
	}

	// Erase the spinner line so the caller's result message replaces it.
	fmt.Fprint(os.Stdout, "\r\x1b[2K")

	if m, ok := finalModel.(*loadingModel); ok {
		return m.completed, m.error, nil
	}
	return false, nil, nil
}

func (m *loadingModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tickStatusUpdate())
}

func (m *loadingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			if m.config.OnEarlyExit != nil {
				m.config.OnEarlyExit(m.olmClient)
			}
			m.completed = false
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case statusUpdateMsg:
		isRunning := m.olmClient.IsRunning()

		if isRunning {
			status, err := m.olmClient.GetStatus()
			if err == nil {
				m.status = status

				// Check for errors before registration
				if status.Error != nil && !status.Registered {
					m.error = status.Error
					if m.config.OnError != nil {
						m.config.OnError(m.olmClient, status.Error)
					}
					m.completed = false
					m.quitting = true
					return m, tea.Quit
				}
			}
		} else {
			m.status = nil
		}

		if m.config.ExitCondition != nil {
			shouldExit, completed := m.config.ExitCondition(m.olmClient, m.status)
			if shouldExit {
				m.completed = completed
				m.quitting = true
				return m, tea.Quit
			}
		}
		return m, tickStatusUpdate()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *loadingModel) View() string {
	if m.quitting {
		return ""
	}
	statusText := m.config.StatusFormatter(m.olmClient.IsRunning(), m.status)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color(logger.ColorLightGray))
	return fmt.Sprintf("%s %s %s", m.spinner.View(), m.config.Header, muted.Render("· "+statusText))
}

type statusUpdateMsg struct{}

func tickStatusUpdate() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return statusUpdateMsg{}
	})
}
