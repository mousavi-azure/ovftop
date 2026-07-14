// Package logs implements the F8 Logs screen: a scrollable view of the
// application's own recent activity (connections, deployments, errors).
package logs

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/logging"
	"github.com/mousavi-azure/ovftop/internal/tui/components"
	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

// ClosedMsg is emitted when the user backs out of the logs screen.
type ClosedMsg struct{}

const maxLines = 1000

// Model is the log viewer screen.
type Model struct {
	styles theme.Styles
	logger *logging.Logger

	view viewport.Model
	err  error

	width, height int
}

// New creates the logs screen and loads the current tail of the log file.
func New(styles theme.Styles, logger *logging.Logger) *Model {
	m := &Model{styles: styles, logger: logger, view: viewport.New(80, 20)}
	m.Reload()
	return m
}

// Reload re-reads the tail of the log file.
func (m *Model) Reload() {
	lines, err := m.logger.Tail(maxLines)
	if err != nil {
		m.err = err
		return
	}
	m.err = nil
	if len(lines) == 0 {
		m.view.SetContent(m.styles.TreeMuted.Render("(no log entries yet)"))
		return
	}

	styled := make([]string, len(lines))
	for i, l := range lines {
		switch {
		case strings.Contains(l, "[ERROR]"):
			styled[i] = m.styles.Error.Render(l)
		case strings.Contains(l, "[WARN]"):
			styled[i] = m.styles.Warning.Render(l)
		default:
			styled[i] = m.styles.TreeItem.Render(l)
		}
	}
	m.view.SetContent(strings.Join(styled, "\n"))
	m.view.GotoBottom()
}

// SetSize updates the viewport dimensions.
func (m *Model) SetSize(w, h int) {
	m.width, m.height = w, h
	m.view.Width = w - 4
	m.view.Height = h - 6
}

// Init satisfies the tea.Model-shaped contract used by the parent app.
func (m *Model) Init() tea.Cmd { return nil }

// Update handles a message and returns the (possibly replaced) model plus
// any command, mirroring tea.Model.Update for use as a sub-model.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc", "q":
		return m, func() tea.Msg { return ClosedMsg{} }
	case "r":
		m.Reload()
		return m, nil
	case "up", "k":
		m.view.LineUp(1)
	case "down", "j":
		m.view.LineDown(1)
	case "pgup":
		m.view.PageUp()
	case "pgdown", " ":
		m.view.PageDown()
	case "g":
		m.view.GotoTop()
	case "G":
		m.view.GotoBottom()
	}
	return m, nil
}

// View renders the logs screen.
func (m *Model) View() string {
	titleBar := components.RenderTitleBar(m.styles, m.width, "📜", "Logs", "")

	body := m.view.View()
	if m.err != nil {
		body = m.styles.Error.Render("⚠ " + m.err.Error())
	}
	panel := m.styles.PanelFocused.Width(m.width - 2).Height(m.height - 4).Render(body)

	help := m.styles.StatusBar.Render("↑/↓ scroll   PgUp/PgDn page   g/G top/bottom   r reload   Esc back")

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, panel, help)
}
