package notice

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fosrl/cli/internal/config"
	"github.com/fosrl/cli/internal/logger"
	"github.com/fosrl/cli/internal/version"
)

var (
	bannerBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(logger.ColorInfo)).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	bannerTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color(logger.ColorWarning))

	bannerBodyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

// Notice is a one-time user message shown on the first eligible CLI invocation.
type Notice struct {
	ID         string
	MinVersion string
	MaxVersion string
	Condition  func(cfg *config.Config) bool
	Lines      func(cfg *config.Config) []string
}

// ShowPending prints registered notices that have not been shown yet and match their condition.
func ShowPending(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}

	state, err := loadState()
	if err != nil {
		return err
	}

	changed := false
	for _, notice := range registeredNotices {
		if state.wasShown(notice.ID) {
			continue
		}
		if !noticeMatchesVersion(notice) {
			continue
		}
		if notice.Condition != nil && !notice.Condition(cfg) {
			continue
		}

		printBanner(notice.Lines(cfg))
		state.markShown(notice.ID)
		changed = true
	}

	if !changed {
		return nil
	}
	return saveState(state)
}

func printBanner(lines []string) {
	if len(lines) == 0 {
		return
	}

	title := lines[0]
	body := strings.Join(lines[1:], "\n")

	var content strings.Builder
	content.WriteString(bannerTitleStyle.Render(title))
	if body != "" {
		content.WriteString("\n\n")
		content.WriteString(bannerBodyStyle.Render(body))
	}

	fmt.Println()
	fmt.Println(bannerBoxStyle.Render(content.String()))
	fmt.Println()
}

func noticeMatchesVersion(notice Notice) bool {
	if notice.MinVersion == "" && notice.MaxVersion == "" {
		return true
	}

	current := version.Version
	if notice.MinVersion != "" {
		cmp, err := version.CompareVersions(current, notice.MinVersion)
		if err != nil || cmp < 0 {
			return false
		}
	}
	if notice.MaxVersion != "" {
		cmp, err := version.CompareVersions(current, notice.MaxVersion)
		if err != nil || cmp > 0 {
			return false
		}
	}
	return true
}
