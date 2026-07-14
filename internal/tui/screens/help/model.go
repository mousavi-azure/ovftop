// Package help implements the F1 Help screen: a static, scrollable
// reference for every keybinding in the app.
package help

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/tui/components"
	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

// ClosedMsg is emitted when the user backs out of the help screen.
type ClosedMsg struct{}

type section struct {
	title string
	rows  [][2]string // key/description pairs
}

var sections = []section{
	{
		title: "Global",
		rows: [][2]string{
			{"F1", "Show this help screen"},
			{"F2", "Disconnect and return to the connection screen"},
			{"F3 / Ctrl+R", "Refresh inventory now"},
			{"F4 / Ctrl+D", "Open the Deploy wizard (requires a selected/any inventory)"},
			{"F5", "Clone the selected VM/template"},
			{"F6", "Export the selected VM/template to an OVA file"},
			{"F7", "Open Settings (theme, auto-refresh interval)"},
			{"F8", "Open the Logs viewer"},
			{"F9", "Open the pull-down Menu"},
			{"F10", "Quit OVFTOP"},
			{"Ctrl+C", "Quit immediately from anywhere"},
		},
	},
	{
		title: "Infrastructure tree (dashboard)",
		rows: [][2]string{
			{"↑ / k", "Move selection up"},
			{"↓ / j", "Move selection down"},
			{"Enter / Space", "Expand or collapse the selected node"},
		},
	},
	{
		title: "Clone prompt (F5)",
		rows: [][2]string{
			{"Enter", "Confirm and start cloning"},
			{"Esc", "Cancel"},
		},
	},
	{
		title: "Logs screen (F8)",
		rows: [][2]string{
			{"↑/↓, PgUp/PgDn", "Scroll"},
			{"g / G", "Jump to top / bottom"},
			{"r", "Reload from disk"},
			{"Esc / q", "Back to dashboard"},
		},
	},
	{
		title: "Menu & Settings (F9 / F7)",
		rows: [][2]string{
			{"↑/↓", "Select an item"},
			{"Enter", "Activate / cycle the selected item"},
			{"Esc", "Back"},
		},
	},
}

// Model is the help screen.
type Model struct {
	styles theme.Styles
	view   viewport.Model

	width, height int
}

// New creates the help screen with its content already rendered.
func New(styles theme.Styles) *Model {
	m := &Model{styles: styles, view: viewport.New(80, 20)}
	m.view.SetContent(m.render())
	return m
}

func (m *Model) render() string {
	var blocks []string
	for _, sec := range sections {
		keyWidth := 0
		for _, r := range sec.rows {
			if len(r[0]) > keyWidth {
				keyWidth = len(r[0])
			}
		}
		var rows []string
		for _, r := range sec.rows {
			key := r[0] + strings.Repeat(" ", keyWidth-len(r[0]))
			rows = append(rows, m.styles.Success.Render(key)+"   "+m.styles.TreeItem.Render(r[1]))
		}
		blocks = append(blocks,
			m.styles.PanelTitle.Render(sec.title),
			strings.Join(rows, "\n"),
			"",
		)
	}
	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
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
	case "esc", "q", "f1":
		return m, func() tea.Msg { return ClosedMsg{} }
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

// View renders the help screen.
func (m *Model) View() string {
	titleBar := components.RenderTitleBar(m.styles, m.width, "❓", "Help", "")
	panel := m.styles.PanelFocused.Width(m.width - 2).Height(m.height - 4).Render(m.view.View())
	help := m.styles.StatusBar.Render("↑/↓ scroll   PgUp/PgDn page   Esc back")
	return lipgloss.JoinVertical(lipgloss.Left, titleBar, panel, help)
}
