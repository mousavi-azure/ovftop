// Package connect implements the connection wizard screen: a list of
// saved ESXi/vCenter profiles plus a form for adding, editing, and
// authenticating against new ones.
package connect

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/config"
	"github.com/mousavi-azure/ovftop/internal/tui/components"
	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

type mode int

const (
	modeList mode = iota
	modeForm
)

// field identifies one focusable control in the form. Text fields index
// into Model.inputs; the rest are handled specially.
type field int

const (
	fieldName field = iota
	fieldDescription
	fieldType
	fieldHostname
	fieldUsername
	fieldPassword
	fieldPort
	fieldIgnoreSSL
	fieldRemember
	fieldSubmit
	fieldCount
)

// textFieldIndex maps a field to its position in Model.inputs, or -1 if
// the field isn't a text input.
func textFieldIndex(f field) int {
	switch f {
	case fieldName:
		return 0
	case fieldDescription:
		return 1
	case fieldHostname:
		return 2
	case fieldUsername:
		return 3
	case fieldPassword:
		return 4
	case fieldPort:
		return 5
	default:
		return -1
	}
}

// ConnectRequestedMsg is emitted when the user submits the form (or picks
// an already-authenticated saved profile) and wants to connect now. The
// parent app owns actually dialing out to vSphere.
type ConnectRequestedMsg struct {
	Profile  config.ConnectionProfile
	Password string
}

// Model is the connection wizard screen.
type Model struct {
	styles theme.Styles
	cfg    *config.Manager

	mode       mode
	profiles   []config.ConnectionProfile
	listCursor int

	editingID string
	inputs    []textinput.Model
	connType  config.ConnectionType
	ignoreSSL bool
	remember  bool
	focus     field

	err error

	width, height int
}

// New creates the connection wizard, loading saved profiles from cfg.
func New(styles theme.Styles, cfg *config.Manager) *Model {
	m := &Model{styles: styles, cfg: cfg, mode: modeList}
	m.profiles = cfg.Profiles()
	return m
}

// SetSize updates the viewport dimensions.
func (m *Model) SetSize(w, h int) { m.width, m.height = w, h }

// SetError displays an error banner above the screen (e.g. a failed
// connection attempt reported back by the parent app).
func (m *Model) SetError(err error) { m.err = err }

func (m *Model) resetForm(profile config.ConnectionProfile) {
	m.editingID = profile.ID
	m.connType = profile.Type
	if m.connType == "" {
		m.connType = config.ConnectionESXi
	}
	m.ignoreSSL = profile.IgnoreSSL
	m.remember = profile.Remember

	newInput := func(placeholder string, value string, password bool) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.SetValue(value)
		ti.CharLimit = 128
		if password {
			ti.EchoMode = textinput.EchoPassword
			ti.EchoCharacter = '•'
		}
		return ti
	}

	port := ""
	if profile.Port != 0 {
		port = strconv.Itoa(profile.Port)
	}

	m.inputs = []textinput.Model{
		newInput("e.g. Lab ESXi", profile.Name, false),
		newInput("e.g. Production vCenter, rack 3", profile.Description, false),
		newInput("esxi01.lab.local", profile.Hostname, false),
		newInput("root", profile.Username, false),
		newInput("(leave blank to keep saved password)", "", true),
		newInput("443", port, false),
	}
	m.focus = fieldName
	m.inputs[0].Focus()
}

// Init satisfies the tea.Model-shaped contract used by the parent app.
func (m *Model) Init() tea.Cmd { return textinput.Blink }

// Update handles a message and returns the (possibly replaced) model plus
// any command, mirroring tea.Model.Update for use as a sub-model.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.mode == modeList {
			return m.updateList(msg)
		}
		return m.updateForm(msg)
	}
	return m, nil
}

func (m *Model) updateList(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.listCursor > 0 {
			m.listCursor--
		}
	case "down", "j":
		if m.listCursor < len(m.profiles) {
			m.listCursor++
		}
	case "n":
		m.err = nil
		m.resetForm(config.ConnectionProfile{Type: config.ConnectionESXi, Port: 443})
		m.mode = modeForm
		return m, textinput.Blink
	case "e":
		if m.listCursor < len(m.profiles) {
			m.err = nil
			m.resetForm(m.profiles[m.listCursor])
			m.mode = modeForm
			return m, textinput.Blink
		}
	case "d":
		if m.listCursor < len(m.profiles) {
			p := m.profiles[m.listCursor]
			_ = m.cfg.DeleteProfile(p.ID)
			m.profiles = m.cfg.Profiles()
			if m.listCursor >= len(m.profiles) && m.listCursor > 0 {
				m.listCursor--
			}
		}
	case "enter":
		m.err = nil
		if m.listCursor == len(m.profiles) {
			// "+ New Connection" row.
			m.resetForm(config.ConnectionProfile{Type: config.ConnectionESXi, Port: 443})
			m.mode = modeForm
			return m, textinput.Blink
		}
		p := m.profiles[m.listCursor]
		if pw, ok, _ := m.cfg.Password(p.ID); ok {
			return m, func() tea.Msg { return ConnectRequestedMsg{Profile: p, Password: pw} }
		}
		// No saved password: open the form pre-filled so the user can
		// supply credentials before connecting.
		m.resetForm(p)
		m.mode = modeForm
		return m, textinput.Blink
	}
	return m, nil
}

func (m *Model) updateForm(msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeList
		m.err = nil
		return m, nil
	case "tab", "down":
		m.advanceFocus(1)
		return m, m.focusCmd()
	case "shift+tab", "up":
		m.advanceFocus(-1)
		return m, m.focusCmd()
	case "left", "right":
		switch m.focus {
		case fieldType:
			if m.connType == config.ConnectionESXi {
				m.connType = config.ConnectionVCenter
			} else {
				m.connType = config.ConnectionESXi
			}
			return m, nil
		}
	case " ":
		switch m.focus {
		case fieldIgnoreSSL:
			m.ignoreSSL = !m.ignoreSSL
			return m, nil
		case fieldRemember:
			m.remember = !m.remember
			return m, nil
		}
	case "enter":
		switch m.focus {
		case fieldType:
			if m.connType == config.ConnectionESXi {
				m.connType = config.ConnectionVCenter
			} else {
				m.connType = config.ConnectionESXi
			}
			return m, nil
		case fieldIgnoreSSL:
			m.ignoreSSL = !m.ignoreSSL
			return m, nil
		case fieldRemember:
			m.remember = !m.remember
			return m, nil
		case fieldSubmit:
			return m.submit()
		default:
			m.advanceFocus(1)
			return m, m.focusCmd()
		}
	}

	if idx := textFieldIndex(m.focus); idx >= 0 {
		var cmd tea.Cmd
		m.inputs[idx], cmd = m.inputs[idx].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) advanceFocus(dir int) {
	next := int(m.focus) + dir
	if next < 0 {
		next = int(fieldCount) - 1
	}
	if next >= int(fieldCount) {
		next = 0
	}
	m.focus = field(next)
}

func (m *Model) focusCmd() tea.Cmd {
	for i := range m.inputs {
		m.inputs[i].Blur()
	}
	if idx := textFieldIndex(m.focus); idx >= 0 {
		return m.inputs[idx].Focus()
	}
	return nil
}

func (m *Model) submit() (*Model, tea.Cmd) {
	name := m.inputs[textFieldIndex(fieldName)].Value()
	description := m.inputs[textFieldIndex(fieldDescription)].Value()
	hostname := m.inputs[textFieldIndex(fieldHostname)].Value()
	username := m.inputs[textFieldIndex(fieldUsername)].Value()
	password := m.inputs[textFieldIndex(fieldPassword)].Value()
	portStr := m.inputs[textFieldIndex(fieldPort)].Value()

	if name == "" || hostname == "" || username == "" {
		m.err = fmt.Errorf("name, hostname, and username are required")
		return m, nil
	}

	port := 443
	if portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	profile := config.ConnectionProfile{
		ID:          m.editingID,
		Name:        name,
		Description: description,
		Type:        m.connType,
		Hostname:    hostname,
		Username:    username,
		Port:        port,
		IgnoreSSL:   m.ignoreSSL,
		Remember:    m.remember,
	}

	saved, err := m.cfg.SaveProfile(profile, password)
	if err != nil {
		m.err = err
		return m, nil
	}
	m.profiles = m.cfg.Profiles()

	effectivePassword := password
	if effectivePassword == "" {
		if pw, ok, _ := m.cfg.Password(saved.ID); ok {
			effectivePassword = pw
		}
	}
	if effectivePassword == "" {
		m.err = fmt.Errorf("password is required")
		return m, nil
	}

	m.mode = modeList
	m.err = nil
	return m, func() tea.Msg { return ConnectRequestedMsg{Profile: saved, Password: effectivePassword} }
}

// View renders the current mode.
func (m *Model) View() string {
	if m.mode == modeForm {
		titleBar := components.RenderTitleBar(m.styles, m.width, "🖥", "", "")
		return lipgloss.JoinVertical(lipgloss.Left, titleBar, m.viewForm())
	}

	banner := components.RenderBanner(m.styles, m.width)
	return lipgloss.JoinVertical(lipgloss.Left, banner, "", m.viewList())
}

func typeIcon(t config.ConnectionType) string {
	if t == config.ConnectionVCenter {
		return "☁ "
	}
	return "🖥 "
}

// selectedRowStyle is this screen's own take on "selected row": pure white
// text on the theme's accent background, rather than the shared
// TreeSelected style's dark-on-accent (which reads as flat gray on Dark's
// blue highlight). Kept local to the connect screen rather than changed in
// the shared theme, since the dashboard's tree selection elsewhere isn't
// meant to change.
func selectedRowStyle(styles theme.Styles) lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("255")).
		Background(styles.Palette.Highlight)
}

func (m *Model) viewList() string {
	title := m.styles.PanelTitle.Render("🔌 Saved Connections")
	selected := selectedRowStyle(m.styles)

	var rows []string
	for i, p := range m.profiles {
		label := fmt.Sprintf("%s %s   👤 %s@%s   [%s]", typeIcon(p.Type), p.Name, p.Username, p.Hostname, p.Type)
		style := m.styles.TreeItem
		descStyle := m.styles.TreeMuted
		if i == m.listCursor {
			style = selected
			descStyle = selected
		}
		entry := style.Render(label)
		if p.Description != "" {
			entry = lipgloss.JoinVertical(lipgloss.Left, entry, descStyle.Render("     📝 "+p.Description))
		}
		rows = append(rows, entry, "")
	}
	newRow := "➕ New Connection"
	if m.listCursor == len(m.profiles) {
		newRow = selected.Render(newRow)
	} else {
		newRow = m.styles.Success.Render(newRow)
	}
	rows = append(rows, newRow)

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	if len(m.profiles) == 0 {
		body = lipgloss.JoinVertical(lipgloss.Left,
			m.styles.TreeMuted.Render("📭 No saved connections yet."), "", body)
	}
	if m.err != nil {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", m.styles.Error.Render("⚠ "+m.err.Error()))
	}

	help := m.styles.StatusBar.Render("↑/↓ select   ⏎ connect   n new   e edit   d delete   Ctrl+C quit")

	panel := m.styles.PanelFocused.Width(m.width - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", body),
	)

	return lipgloss.JoinVertical(lipgloss.Left, panel, help)
}

func (m *Model) viewForm() string {
	label := func(f field, text string) string {
		if m.focus == f {
			return m.styles.PanelTitle.Render("▸ " + text)
		}
		return m.styles.InputLabel.Render("  " + text)
	}

	row := func(f field, text string) string {
		idx := textFieldIndex(f)
		style := m.styles.InputBox
		if m.focus == f {
			style = m.styles.InputFocus
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, label(f, text)+" ", style.Render(m.inputs[idx].View()))
	}

	typeVal := "🖥 ESXi"
	if m.connType == config.ConnectionVCenter {
		typeVal = "☁ vCenter"
	}
	typeStyle := m.styles.InputBox
	if m.focus == fieldType {
		typeStyle = m.styles.InputFocus
	}
	typeRow := lipgloss.JoinHorizontal(lipgloss.Top, label(fieldType, "🔌 Connection Type")+" ", typeStyle.Render("◂ "+typeVal+" ▸"))

	checkbox := func(f field, text string, checked bool) string {
		box := "☐"
		if checked {
			box = "☑"
		}
		line := box + " " + text
		if m.focus == f {
			return m.styles.PanelTitle.Render("▸ " + line)
		}
		return m.styles.InputLabel.Render("  " + line)
	}

	submitLabel := "  🚀 Connect"
	if m.focus == fieldSubmit {
		submitLabel = m.styles.Success.Render("▸ 🚀 Connect")
	}

	title := "➕ New Connection"
	if m.editingID != "" {
		title = "✏ Edit Connection"
	}

	lines := []string{
		m.styles.PanelTitle.Render(title),
		"",
		row(fieldName, "🏷 Name"),
		row(fieldDescription, "📝 Description"),
		typeRow,
		row(fieldHostname, "🌐 Hostname"),
		row(fieldUsername, "👤 Username"),
		row(fieldPassword, "🔑 Password"),
		row(fieldPort, "🔢 Port"),
		"",
		checkbox(fieldIgnoreSSL, "Ignore SSL certificate errors", m.ignoreSSL),
		checkbox(fieldRemember, "Remember credentials", m.remember),
		"",
		submitLabel,
	}

	if m.err != nil {
		lines = append(lines, "", m.styles.Error.Render(m.err.Error()))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, lines...)
	panel := m.styles.PanelFocused.Width(m.width - 2).Render(body)
	help := m.styles.StatusBar.Render("Tab/↑↓ next field   ←/→ toggle   Space check   Enter submit   Esc cancel")

	return lipgloss.JoinVertical(lipgloss.Left, panel, help)
}
