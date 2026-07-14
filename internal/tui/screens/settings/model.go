// Package settings implements the F7 Settings screen: theme selection,
// auto-refresh cadence, and read-only display of on-disk config locations.
package settings

import (
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/tui/components"
	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return string(unicode.ToUpper(r[0])) + strings.ToLower(string(r[1:]))
}

// ClosedMsg is emitted when the user backs out of the settings screen.
type ClosedMsg struct{}

// CycleThemeMsg asks the parent app to switch to the next theme.
type CycleThemeMsg struct{}

// SetRefreshIntervalMsg asks the parent app to change the dashboard's
// auto-refresh cadence. Seconds <= 0 means "off".
type SetRefreshIntervalMsg struct{ Seconds int }

var intervalChoices = []int{0, 60, 120, 300, 600}
var intervalLabels = []string{"Off", "1 minute", "2 minutes", "5 minutes", "10 minutes"}

func intervalIndex(seconds int) int {
	for i, s := range intervalChoices {
		if s == seconds {
			return i
		}
	}
	return 2 // default: 2 minutes
}

// Model is the settings screen.
type Model struct {
	styles theme.Styles

	themeName   string
	intervalIdx int
	configDir   string
	logPath     string
	cursor      int

	width, height int
}

// New creates the settings screen showing the current preferences.
func New(styles theme.Styles, currentTheme string, refreshIntervalSeconds int, configDir, logPath string) *Model {
	return &Model{
		styles:      styles,
		themeName:   currentTheme,
		intervalIdx: intervalIndex(refreshIntervalSeconds),
		configDir:   configDir,
		logPath:     logPath,
	}
}

// SetSize updates the viewport dimensions.
func (m *Model) SetSize(w, h int) { m.width, m.height = w, h }

// Init satisfies the tea.Model-shaped contract used by the parent app.
func (m *Model) Init() tea.Cmd { return nil }

const numRows = 2

// Update handles a message and returns the (possibly replaced) model plus
// any command, mirroring tea.Model.Update for use as a sub-model.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		return m, func() tea.Msg { return ClosedMsg{} }
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < numRows-1 {
			m.cursor++
		}
	case "enter", "right", "l":
		return m.activate(1)
	case "left", "h":
		return m.activate(-1)
	}
	return m, nil
}

func (m *Model) activate(dir int) (*Model, tea.Cmd) {
	switch m.cursor {
	case 0:
		return m, func() tea.Msg { return CycleThemeMsg{} }
	case 1:
		n := len(intervalChoices)
		m.intervalIdx = ((m.intervalIdx+dir)%n + n) % n
		seconds := intervalChoices[m.intervalIdx]
		return m, func() tea.Msg { return SetRefreshIntervalMsg{Seconds: seconds} }
	}
	return m, nil
}

// View renders the settings screen.
func (m *Model) View() string {
	titleBar := components.RenderTitleBar(m.styles, m.width, "⚙", "Settings", "")

	row := func(i int, icon, label, value string) string {
		line := icon + "  " + label + ": " + value
		if i == m.cursor {
			return m.styles.PanelTitle.Render("▸ " + line)
		}
		return m.styles.TreeItem.Render("  " + line)
	}

	rows := []string{
		row(0, "🎨", "Theme", capitalize(m.themeName)+"  (Enter/←/→ to cycle: Dark, Light, Dracula, Nord)"),
		row(1, "⏱", "Auto-refresh", intervalLabels[m.intervalIdx]+"  (Enter/←/→ to cycle)"),
		"",
		m.styles.PanelTitle.Render("Storage locations"),
		m.styles.TreeMuted.Render("📁 Config & profiles: " + m.configDir),
		m.styles.TreeMuted.Render("📜 Log file: " + m.logPath),
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	panel := m.styles.PanelFocused.Width(m.width - 2).Render(body)
	help := m.styles.StatusBar.Render("↑/↓ select   Enter/←/→ change   Esc back")

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, panel, help)
}
