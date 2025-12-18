package tui

import (
	"bufio"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/olm"
)

// ExitCondition is a function that determines if the preview should exit
// Returns true if should exit, and a completion status (true = success, false = user cancelled)
type ExitCondition func(client *olm.Client, status *olm.StatusResponse) (shouldExit bool, completed bool)

// StatusFormatter is a function that formats the status display
type StatusFormatter func(isRunning bool, status *olm.StatusResponse) string

// LogPreviewConfig configures the log preview TUI
type LogPreviewConfig struct {
	LogFile         string
	Header          string
	ExitCondition   ExitCondition
	OnEarlyExit     func(client *olm.Client) // Called when user exits early (Ctrl+C)
	StatusFormatter StatusFormatter          // Status formatter (required)
}

// logPreviewModel is the bubbletea model for the live log preview
type logPreviewModel struct {
	config        LogPreviewConfig
	olmClient     *olm.Client
	logLines      []string
	status        *olm.StatusResponse
	lastLogPos    int64
	completedTime *time.Time
	completed     bool
	width         int
}

// NewLogPreview creates and runs a new log preview TUI
func NewLogPreview(config LogPreviewConfig) (completed bool, err error) {
	model := &logPreviewModel{
		config:    config,
		olmClient: olm.NewClient(""),
		logLines:  []string{},
	}

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return false, err
	}

	if previewModel, ok := finalModel.(*logPreviewModel); ok {
		return previewModel.completed, nil
	}
	return false, nil
}

// Init initializes the model
func (m *logPreviewModel) Init() tea.Cmd {
	// Start with initial delay, then start ticking
	return tea.Batch(
		tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
			// After delay, start the tickers
			return initCompleteMsg{}
		}),
		tickStatusUpdate(),
	)
}

// Update handles messages
func (m *logPreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			// User exited early - call callback if provided
			if m.config.OnEarlyExit != nil {
				m.config.OnEarlyExit(m.olmClient)
			}
			m.completed = false
			return m, tea.Quit
		}
		return m, nil

	case initCompleteMsg:
		// Start log updates after initial delay
		return m, tickLogUpdate()

	case logUpdateMsg:
		// Update log lines
		lines, newPos := getLastLogLines(m.config.LogFile, 5, m.lastLogPos)
		if newPos != m.lastLogPos {
			m.lastLogPos = newPos
			m.logLines = lines
		}
		return m, tickLogUpdate()

	case statusUpdateMsg:
		// Update status
		isRunning := m.olmClient.IsRunning()

		if isRunning {
			status, err := m.olmClient.GetStatus()
			if err == nil {
				m.status = status
			}
		} else {
			// Socket doesn't exist - clear status
			m.status = nil
		}

		// Check exit condition (pass current status, even if nil)
		if m.config.ExitCondition != nil {
			shouldExit, completed := m.config.ExitCondition(m.olmClient, m.status)
			if shouldExit {
				if m.completedTime == nil {
					now := time.Now()
					m.completedTime = &now
					m.completed = completed
				} else if time.Since(*m.completedTime) >= 1*time.Second {
					// Wait 1 second, then exit
					return m, tea.Quit
				}
			} else {
				// Reset completed time if condition no longer met
				m.completedTime = nil
			}
		}
		return m, tickStatusUpdate()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	}
	return m, nil
}

// View renders the model
func (m *logPreviewModel) View() string {
	var sb strings.Builder

	// Styles
	logStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(logger.ColorLightGray))

	// Header
	sb.WriteString(m.config.Header)
	sb.WriteString("\n")

	// Log lines (last 5, always show 5 lines)
	for i := 0; i < 5; i++ {
		if i < len(m.logLines) {
			line := m.logLines[i]
			// Truncate long lines
			if len(line) > 80 {
				line = line[:77] + "..."
			}
			sb.WriteString(logStyle.Render(line))
		}
		sb.WriteString("\n")
	}

	// Status line
	sb.WriteString("Status: ")
	sb.WriteString(m.config.StatusFormatter(m.olmClient.IsRunning(), m.status))

	return sb.String()
}

// Messages for bubbletea
type (
	logUpdateMsg    struct{}
	statusUpdateMsg struct{}
	initCompleteMsg struct{}
)

// tickLogUpdate sends a log update tick
func tickLogUpdate() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return logUpdateMsg{}
	})
}

// tickStatusUpdate sends a status update tick
func tickStatusUpdate() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(t time.Time) tea.Msg {
		return statusUpdateMsg{}
	})
}

// getLastLogLines reads the last N lines from a log file starting from a position
func getLastLogLines(logPath string, n int, lastPos int64) ([]string, int64) {
	file, err := os.Open(logPath)
	if err != nil {
		return []string{}, lastPos
	}
	defer file.Close()

	// Get file size
	info, err := file.Stat()
	if err != nil {
		return []string{}, lastPos
	}

	fileSize := info.Size()

	// If file hasn't grown, return empty
	if fileSize <= lastPos {
		return []string{}, lastPos
	}

	// Read from lastPos to end
	file.Seek(lastPos, io.SeekStart)
	scanner := bufio.NewScanner(file)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	// Return last N lines
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	return lines, fileSize
}
