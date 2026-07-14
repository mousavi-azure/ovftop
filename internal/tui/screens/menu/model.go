// Package menu implements the F9 pull-down menu screen: a discoverable
// list of app-wide actions (view logs, switch theme, refresh, disconnect,
// quit) plus an inline About panel.
package menu

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

type actionKind int

const (
	actionViewLogs actionKind = iota
	actionCycleTheme
	actionRefreshNow
	actionDisconnect
	actionAbout
	actionQuit
)

type item struct {
	icon, label, desc string
	kind              actionKind
}

// ClosedMsg is emitted when the user backs out of the menu.
type ClosedMsg struct{}

// ViewLogsMsg asks the parent app to open the Logs screen.
type ViewLogsMsg struct{}

// CycleThemeMsg asks the parent app to switch to the next theme.
type CycleThemeMsg struct{}

// RefreshNowMsg asks the parent app to re-run discovery immediately.
type RefreshNowMsg struct{}

// DisconnectMsg asks the parent app to return to the connection screen.
type DisconnectMsg struct{}

// QuitMsg asks the parent app to exit.
type QuitMsg struct{}

// Model is the pull-down menu screen.
type Model struct {
	styles theme.Styles
	items  []item
	cursor int

	showAbout bool

	width, height int
}

// New creates the menu, labeling the theme entry with the current theme
// name so it's obvious what cycling it will do.
func New(styles theme.Styles, currentTheme string) *Model {
	return &Model{
		styles: styles,
		items: []item{
			{"📜", "View Logs", "Browse recent connection/deploy activity", actionViewLogs},
			{"🎨", "Switch Theme", "Current: " + capitalize(currentTheme) + " — cycles Dark/Light/Dracula/Nord", actionCycleTheme},
			{"🔄", "Refresh Now", "Re-run inventory discovery immediately", actionRefreshNow},
			{"🔌", "Disconnect", "Return to the connection screen", actionDisconnect},
			{"ℹ", "About", "Show credits and version info", actionAbout},
			{"🚪", "Quit", "Exit OVFTOP", actionQuit},
		},
	}
}

// SetSize updates the viewport dimensions.
func (m *Model) SetSize(w, h int) { m.width, m.height = w, h }

// Init satisfies the tea.Model-shaped contract used by the parent app.
func (m *Model) Init() tea.Cmd { return nil }

// Update handles a message and returns the (possibly replaced) model plus
// any command, mirroring tea.Model.Update for use as a sub-model.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.showAbout {
		if keyMsg.String() == "esc" || keyMsg.String() == "enter" {
			m.showAbout = false
		}
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
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}
	case "enter":
		return m.trigger()
	}
	return m, nil
}

func (m *Model) trigger() (*Model, tea.Cmd) {
	switch m.items[m.cursor].kind {
	case actionViewLogs:
		return m, func() tea.Msg { return ViewLogsMsg{} }
	case actionCycleTheme:
		return m, func() tea.Msg { return CycleThemeMsg{} }
	case actionRefreshNow:
		return m, func() tea.Msg { return RefreshNowMsg{} }
	case actionDisconnect:
		return m, func() tea.Msg { return DisconnectMsg{} }
	case actionAbout:
		m.showAbout = true
		return m, nil
	case actionQuit:
		return m, func() tea.Msg { return QuitMsg{} }
	}
	return m, nil
}

// View renders the menu screen.
func (m *Model) View() string {
	titleBar := components.RenderTitleBar(m.styles, m.width, "📋", "Menu", "")

	var body string
	if m.showAbout {
		body = lipgloss.JoinVertical(lipgloss.Center,
			components.RenderBanner(m.styles, m.width-4),
			"",
			m.styles.StatusBar.Render("Enter/Esc back to menu"),
		)
	} else {
		var rows []string
		for i, it := range m.items {
			line := it.icon + "  " + it.label
			desc := "     " + it.desc
			if i == m.cursor {
				rows = append(rows,
					m.styles.PanelTitle.Render("▸ "+line),
					m.styles.TreeMuted.Render(desc),
				)
			} else {
				rows = append(rows,
					m.styles.TreeItem.Render("  "+line),
					m.styles.TreeMuted.Render(desc),
				)
			}
			rows = append(rows, "")
		}
		body = lipgloss.JoinVertical(lipgloss.Left, rows...)
	}

	panel := m.styles.PanelFocused.Width(m.width - 2).Render(body)
	help := m.styles.StatusBar.Render("↑/↓ select   Enter choose   Esc back")

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, panel, help)
}
