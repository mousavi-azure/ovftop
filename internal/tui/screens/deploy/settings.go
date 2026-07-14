package deploywizard

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/deploy"
)

type settingsField int

const (
	sfVMName settingsField = iota
	sfCompute
	sfDatastore
	sfDiskMode
	sfPowerOn
	sfTemplate
	sfContinue
)

// visibleSettingsFields returns the ordered, connection-appropriate set
// of focusable fields: standalone ESXi has no compute-resource picker
// (there's only one possible target) and can't mark VMs as templates
// (ovftool: "Template creation only supported on VirtualCenter target
// type" — confirmed against a live target while building this).
func (m *Model) visibleSettingsFields() []settingsField {
	fields := []settingsField{sfVMName}
	if !m.isESXi {
		fields = append(fields, sfCompute)
	}
	fields = append(fields, sfDatastore, sfDiskMode, sfPowerOn)
	if !m.isESXi {
		fields = append(fields, sfTemplate)
	}
	return append(fields, sfContinue)
}

func (m *Model) updateSettings(msg tea.Msg) (*Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	fields := m.visibleSettingsFields()
	current := fields[m.settingsFocus%len(fields)]

	switch keyMsg.String() {
	case "esc":
		m.step = stepMetadata
		return m, nil
	case "tab", "down":
		m.settingsFocus = (m.settingsFocus + 1) % len(fields)
		return m, nil
	case "shift+tab", "up":
		m.settingsFocus = (m.settingsFocus - 1 + len(fields)) % len(fields)
		return m, nil
	case "left", "right":
		delta := 1
		if keyMsg.String() == "left" {
			delta = -1
		}
		switch current {
		case sfCompute:
			if n := len(m.computeOptions); n > 0 {
				m.computeCursor = (m.computeCursor + delta + n) % n
			}
		case sfDatastore:
			if n := len(m.datastoreOptions); n > 0 {
				m.datastoreCursor = (m.datastoreCursor + delta + n) % n
			}
		case sfDiskMode:
			m.diskModeThick = !m.diskModeThick
		}
		return m, nil
	case " ":
		switch current {
		case sfPowerOn:
			m.powerOn = !m.powerOn
		case sfTemplate:
			m.importAsTemplate = !m.importAsTemplate
		}
		return m, nil
	case "enter":
		switch current {
		case sfDiskMode:
			m.diskModeThick = !m.diskModeThick
		case sfPowerOn:
			m.powerOn = !m.powerOn
		case sfTemplate:
			m.importAsTemplate = !m.importAsTemplate
		case sfContinue:
			if err := m.validateSettings(); err != nil {
				m.err = err
				return m, nil
			}
			m.err = nil
			m.step = stepNetwork
		default:
			m.settingsFocus = (m.settingsFocus + 1) % len(fields)
		}
		return m, nil
	}

	if current == sfVMName {
		var cmd tea.Cmd
		m.vmNameInput, cmd = m.vmNameInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) validateSettings() error {
	if m.vmNameInput.Value() == "" {
		return errRequired("VM name")
	}
	if len(m.datastoreOptions) == 0 {
		return errRequired("a datastore (none discovered)")
	}
	if !m.isESXi && len(m.computeOptions) == 0 {
		return errRequired("a target host/cluster (none discovered)")
	}
	return nil
}

func errRequired(what string) error { return &requiredError{what} }

type requiredError struct{ what string }

func (e *requiredError) Error() string { return e.what + " is required" }

func (m *Model) diskMode() deploy.DiskMode {
	if m.diskModeThick {
		return deploy.DiskModeThick
	}
	return deploy.DiskModeThin
}

func (m *Model) viewSettings() string {
	fields := m.visibleSettingsFields()
	current := fields[m.settingsFocus%len(fields)]

	fieldLabel := func(f settingsField, text string) string {
		if current == f {
			return m.styles.PanelTitle.Render("▸ " + text)
		}
		return m.styles.InputLabel.Render("  " + text)
	}

	box := func(f settingsField, content string) string {
		style := m.styles.InputBox
		if current == f {
			style = m.styles.InputFocus
		}
		return style.Render(content)
	}

	picklist := func(f settingsField, options []string, cursor int) string {
		if len(options) == 0 {
			return box(f, "(none discovered)")
		}
		return box(f, "◂ "+options[cursor]+" ▸")
	}

	computeNames := make([]string, len(m.computeOptions))
	for i, c := range m.computeOptions {
		computeNames[i] = c.Name
	}
	datastoreNames := make([]string, len(m.datastoreOptions))
	for i, d := range m.datastoreOptions {
		datastoreNames[i] = d.Name
	}

	diskModeVal := "🌱 Thin"
	if m.diskModeThick {
		diskModeVal = "🧱 Thick"
	}

	checkbox := func(f settingsField, text string, checked bool, disabled bool) string {
		box := "☐"
		if checked {
			box = "☑"
		}
		line := box + " " + text
		if disabled {
			return m.styles.TreeMuted.Render("  " + line + " (vCenter only)")
		}
		if current == f {
			return m.styles.PanelTitle.Render("▸ " + line)
		}
		return m.styles.InputLabel.Render("  " + line)
	}

	joinRow := func(label, field string) string {
		return lipgloss.JoinHorizontal(lipgloss.Top, label+" ", field)
	}

	rows := []string{
		joinRow(fieldLabel(sfVMName, "🏷 VM Name"), box(sfVMName, m.vmNameInput.View())),
	}
	if !m.isESXi {
		rows = append(rows, joinRow(fieldLabel(sfCompute, "🖧 Target"), picklist(sfCompute, computeNames, m.computeCursor)))
	}
	rows = append(rows,
		joinRow(fieldLabel(sfDatastore, "💽 Datastore"), picklist(sfDatastore, datastoreNames, m.datastoreCursor)),
		joinRow(fieldLabel(sfDiskMode, "💾 Disk Mode"), box(sfDiskMode, "◂ "+diskModeVal+" ▸")),
		"",
		checkbox(sfPowerOn, "Power on after deployment", m.powerOn, false),
	)
	if m.isESXi {
		rows = append(rows, checkbox(sfTemplate, "Mark as template", false, true))
	} else {
		rows = append(rows, checkbox(sfTemplate, "Mark as template", m.importAsTemplate, false))
	}

	submitLabel := "  [ Continue ]"
	if current == sfContinue {
		submitLabel = m.styles.Success.Render("▸ [ Continue ]")
	}
	rows = append(rows, "", submitLabel)

	if m.err != nil {
		rows = append(rows, "", m.styles.Error.Render("⚠ "+m.err.Error()))
	}

	title := m.styles.PanelTitle.Render("Deployment Settings")
	body := lipgloss.JoinVertical(lipgloss.Left, append([]string{title, ""}, rows...)...)
	panel := m.styles.PanelFocused.Width(m.width - 2).Render(body)
	help := m.styles.StatusBar.Render("Tab/↑↓ field   ←/→ change   Space check   Enter continue   Esc back")

	return lipgloss.JoinVertical(lipgloss.Left, m.titleBar(), m.stepper(), panel, help)
}
