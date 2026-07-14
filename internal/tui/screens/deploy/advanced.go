package deploywizard

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// advanced step focus layout: [0..len(propertyInputs)) properties,
// then extra-args, then continue.
func (m *Model) advancedFieldCount() int { return len(m.propertyInputs) + 2 }

func (m *Model) updateAdvanced(msg tea.Msg) (*Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	count := m.advancedFieldCount()
	extraArgsIdx := len(m.propertyInputs)
	continueIdx := len(m.propertyInputs) + 1

	switch keyMsg.String() {
	case "esc":
		m.step = stepNetwork
		return m, nil
	case "tab", "down":
		m.advancedFocus = (m.advancedFocus + 1) % count
		return m, m.focusAdvanced(extraArgsIdx)
	case "shift+tab", "up":
		m.advancedFocus = (m.advancedFocus - 1 + count) % count
		return m, m.focusAdvanced(extraArgsIdx)
	case "enter":
		if m.advancedFocus == continueIdx {
			m.buildSpec()
			m.step = stepSummary
			return m, nil
		}
		m.advancedFocus = (m.advancedFocus + 1) % count
		return m, m.focusAdvanced(extraArgsIdx)
	}

	switch {
	case m.advancedFocus < len(m.propertyInputs):
		var cmd tea.Cmd
		m.propertyInputs[m.advancedFocus], cmd = m.propertyInputs[m.advancedFocus].Update(msg)
		return m, cmd
	case m.advancedFocus == extraArgsIdx:
		var cmd tea.Cmd
		m.extraArgsInput, cmd = m.extraArgsInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) focusAdvanced(extraArgsIdx int) tea.Cmd {
	for i := range m.propertyInputs {
		m.propertyInputs[i].Blur()
	}
	m.extraArgsInput.Blur()

	if m.advancedFocus < len(m.propertyInputs) {
		return m.propertyInputs[m.advancedFocus].Focus()
	}
	if m.advancedFocus == extraArgsIdx {
		return m.extraArgsInput.Focus()
	}
	return nil
}

func (m *Model) viewAdvanced() string {
	title := m.styles.PanelTitle.Render("Advanced Settings")
	extraArgsIdx := len(m.propertyInputs)
	continueIdx := len(m.propertyInputs) + 1

	var rows []string
	if len(m.descriptor.Properties) == 0 {
		rows = append(rows, m.styles.TreeMuted.Render("This package declares no configurable OVF properties."))
	}
	for i, p := range m.descriptor.Properties {
		label := p.Label
		if label == "" {
			label = p.Key
		}
		lstyle := m.styles.InputLabel
		box := m.styles.InputBox
		if m.advancedFocus == i {
			lstyle = m.styles.PanelTitle
			box = m.styles.InputFocus
		}
		prefix := "  "
		if m.advancedFocus == i {
			prefix = "▸ "
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, lstyle.Render(prefix+"⚙ "+label)+" ", box.Render(m.propertyInputs[i].View())))
		if p.Description != "" {
			rows = append(rows, m.styles.TreeMuted.Render("    "+p.Description))
		}
	}

	rows = append(rows, "")
	extraLabel := m.styles.InputLabel.Render("  🧩 Extra ovftool args")
	extraBox := m.styles.InputBox
	if m.advancedFocus == extraArgsIdx {
		extraLabel = m.styles.PanelTitle.Render("▸ 🧩 Extra ovftool args")
		extraBox = m.styles.InputFocus
	}
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, extraLabel+" ", extraBox.Render(m.extraArgsInput.View())))

	submitLabel := "  [ Continue ]"
	if m.advancedFocus == continueIdx {
		submitLabel = m.styles.Success.Render("▸ [ Continue ]")
	}
	rows = append(rows, "", submitLabel)

	body := lipgloss.JoinVertical(lipgloss.Left, append([]string{title, ""}, rows...)...)
	panel := m.styles.PanelFocused.Width(m.width - 2).Render(body)
	help := m.styles.StatusBar.Render("Tab/↑↓ field   Enter continue   Esc back")

	return lipgloss.JoinVertical(lipgloss.Left, m.titleBar(), m.stepper(), panel, help)
}
