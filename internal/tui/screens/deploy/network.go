package deploywizard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) updateNetwork(msg tea.Msg) (*Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	rows := len(m.descriptor.Networks) + 1 // + Continue
	switch keyMsg.String() {
	case "esc":
		m.step = stepSettings
		return m, nil
	case "tab", "down":
		m.networkCursor = (m.networkCursor + 1) % rows
	case "shift+tab", "up":
		m.networkCursor = (m.networkCursor - 1 + rows) % rows
	case "left", "right":
		if m.networkCursor >= len(m.descriptor.Networks) {
			return m, nil
		}
		delta := 1
		if keyMsg.String() == "left" {
			delta = -1
		}
		// Cycle through -1 (unmapped) plus every discovered target.
		n := len(m.networkTargets) + 1
		cur := m.networkMap[m.networkCursor] + 1
		cur = (cur + delta + n) % n
		m.networkMap[m.networkCursor] = cur - 1
	case "enter":
		if m.networkCursor == len(m.descriptor.Networks) {
			if err := m.validateNetworkMap(); err != nil {
				m.err = err
				return m, nil
			}
			m.err = nil
			m.step = stepAdvanced
		}
	}
	return m, nil
}

func (m *Model) validateNetworkMap() error {
	for i, n := range m.descriptor.Networks {
		if m.networkMap[i] == -1 {
			return fmt.Errorf("map a target network for %q first", n.Name)
		}
	}
	return nil
}

func (m *Model) viewNetwork() string {
	title := m.styles.PanelTitle.Render("Network Mapping")

	var rows []string
	if len(m.descriptor.Networks) == 0 {
		rows = append(rows, m.styles.TreeMuted.Render("This package declares no networks."))
	}
	for i, n := range m.descriptor.Networks {
		targetLabel := "-- unmapped --"
		if idx := m.networkMap[i]; idx >= 0 && idx < len(m.networkTargets) {
			targetLabel = m.networkTargets[idx].Name
		}
		left := fmt.Sprintf("🌐 %s", n.Name)
		right := "◂ " + targetLabel + " ▸"

		style := m.styles.InputLabel
		box := m.styles.InputBox
		if m.networkCursor == i {
			style = m.styles.PanelTitle
			box = m.styles.InputFocus
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, style.Render("▸ "+left), "  →  ", box.Render(right)))
	}

	submitLabel := "  [ Continue ]"
	if m.networkCursor == len(m.descriptor.Networks) {
		submitLabel = m.styles.Success.Render("▸ [ Continue ]")
	}
	rows = append(rows, "", submitLabel)

	if m.err != nil {
		rows = append(rows, "", m.styles.Error.Render("⚠ "+m.err.Error()))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, append([]string{title, ""}, rows...)...)
	panel := m.styles.PanelFocused.Width(m.width - 2).Render(body)
	help := m.styles.StatusBar.Render("Tab/↑↓ field   ←/→ change mapping   Enter continue   Esc back")

	return lipgloss.JoinVertical(lipgloss.Left, m.titleBar(), m.stepper(), panel, help)
}
