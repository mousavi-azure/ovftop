// Package dashboard implements the Midnight-Commander-style main screen:
// a top info bar, a left infrastructure tree, a right details panel, and
// a bottom F-key status bar.
package dashboard

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vmware/govmomi/vim25/types"

	"github.com/mousavi-azure/ovftop/internal/tui/components"
	"github.com/mousavi-azure/ovftop/internal/tui/theme"
	"github.com/mousavi-azure/ovftop/internal/vsphere"
)

// defaultRefreshInterval is how often the dashboard re-runs discovery on
// its own, so long-running sessions stay reasonably in sync with the
// connected host/vCenter without the user having to remember F3. It can be
// changed at runtime from the Settings screen via SetRefreshInterval.
const defaultRefreshInterval = 2 * time.Minute

// RefreshRequestedMsg asks the parent app to re-run discovery.
type RefreshRequestedMsg struct{}

type autoRefreshTickMsg struct{}

// DisconnectRequestedMsg asks the parent app to return to the connection
// wizard (F2 — switch/re-connect).
type DisconnectRequestedMsg struct{}

// QuitRequestedMsg asks the parent app to exit.
type QuitRequestedMsg struct{}

// DeployRequestedMsg asks the parent app to open the deploy wizard (F4).
type DeployRequestedMsg struct{}

// CloneRequestedMsg asks the parent app to clone the given VM/template to
// a new VM named NewName (F5, confirmed via the inline name prompt).
type CloneRequestedMsg struct {
	Ref     types.ManagedObjectReference
	NewName string
}

// ExportRequestedMsg asks the parent app to open the export screen (F6)
// for the given VM/template.
type ExportRequestedMsg struct {
	Ref        types.ManagedObjectReference
	Name       string
	Datacenter string
}

// SettingsRequestedMsg asks the parent app to open the settings screen (F7).
type SettingsRequestedMsg struct{}

// LogsRequestedMsg asks the parent app to open the logs screen (F8).
type LogsRequestedMsg struct{}

// MenuRequestedMsg asks the parent app to open the pull-down menu (F9).
type MenuRequestedMsg struct{}

// HelpRequestedMsg asks the parent app to open the help screen (F1).
type HelpRequestedMsg struct{}

// Model is the main dashboard screen.
type Model struct {
	styles theme.Styles
	tree   *components.Tree

	connectionLabel string
	hostLabel       string
	inventory       *vsphere.Inventory
	refreshing      bool
	err             error

	refreshInterval time.Duration

	cloning     bool
	cloneActive bool
	cloneInput  textinput.Model
	cloneRef    types.ManagedObjectReference
	cloneSource string
	cloneErr    error

	width, height         int
	leftWidth, rightWidth int
	bodyHeight            int
}

// New creates an empty dashboard; call SetInventory once discovery
// completes.
func New(styles theme.Styles) *Model {
	ti := textinput.New()
	ti.Prompt = "▸ "
	ti.CharLimit = 80
	return &Model{
		styles:          styles,
		tree:            components.NewTree(styles),
		refreshInterval: defaultRefreshInterval,
		cloneInput:      ti,
	}
}

// SetRefreshInterval changes how often the dashboard auto-refreshes.
// Passing 0 or less disables auto-refresh entirely.
func (m *Model) SetRefreshInterval(d time.Duration) { m.refreshInterval = d }

// RefreshInterval returns the current auto-refresh interval (0 = off).
func (m *Model) RefreshInterval() time.Duration { return m.refreshInterval }

// SetCloning toggles the "cloning…" indicator shown in the info bar and
// clears it once the parent app reports the clone finished.
func (m *Model) SetCloning(v bool) { m.cloning = v }

// SetInventory replaces the displayed inventory snapshot.
func (m *Model) SetInventory(inv *vsphere.Inventory, connectionLabel, hostLabel string) {
	m.inventory = inv
	m.connectionLabel = connectionLabel
	m.hostLabel = hostLabel
	m.tree.SetRoots(inv.Root)
	m.err = nil
	m.refreshing = false
}

// SetRefreshing toggles the "refreshing" indicator shown in the info bar.
func (m *Model) SetRefreshing(v bool) { m.refreshing = v }

// SetError records a background error (e.g. a failed refresh) to display.
func (m *Model) SetError(err error) {
	m.err = err
	m.refreshing = false
}

// SetSize updates the viewport and re-lays-out the tree/details split.
func (m *Model) SetSize(w, h int) {
	m.width, m.height = w, h

	const titleBarHeight = 1
	const infoBarHeight = 4
	const statusBarHeight = 1
	usageBarHeight := m.usageBarRowCount() + 2                                               // panel border
	m.bodyHeight = h - titleBarHeight - infoBarHeight - usageBarHeight - statusBarHeight - 2 // panel borders
	if m.bodyHeight < 1 {
		m.bodyHeight = 1
	}

	m.leftWidth = w * 2 / 5
	m.rightWidth = w - m.leftWidth
	if m.leftWidth < 10 {
		m.leftWidth = 10
	}

	m.tree.SetSize(m.leftWidth-4, m.bodyHeight-2)
}

// Tick schedules the recurring auto-refresh. Call it once when the
// dashboard becomes active. It's a no-op (returns nil) when auto-refresh
// has been turned off from Settings.
func (m *Model) Tick() tea.Cmd {
	if m.refreshInterval <= 0 {
		return nil
	}
	interval := m.refreshInterval
	return tea.Tick(interval, func(time.Time) tea.Msg { return autoRefreshTickMsg{} })
}

// cloneableSelection reports whether the currently selected tree node is a
// VM or template, i.e. something F5/F6 can act on.
func (m *Model) cloneableSelection() *vsphere.Node {
	n := m.tree.Selected()
	if n == nil || (n.Kind != vsphere.KindVM && n.Kind != vsphere.KindTemplate) {
		return nil
	}
	return n
}

// Update handles a message and returns the (possibly replaced) model plus
// any command, mirroring tea.Model.Update for use as a sub-model.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	if _, ok := msg.(autoRefreshTickMsg); ok {
		m.refreshing = true
		return m, tea.Batch(
			func() tea.Msg { return RefreshRequestedMsg{} },
			m.Tick(),
		)
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	if m.cloneActive {
		return m.updateCloneModal(keyMsg)
	}

	switch keyMsg.String() {
	case "up", "k":
		m.tree.MoveUp()
	case "down", "j":
		m.tree.MoveDown()
	case "enter", " ":
		m.tree.ToggleExpand()
	case "f1":
		return m, func() tea.Msg { return HelpRequestedMsg{} }
	case "f2":
		return m, func() tea.Msg { return DisconnectRequestedMsg{} }
	case "f3", "ctrl+r":
		m.refreshing = true
		return m, func() tea.Msg { return RefreshRequestedMsg{} }
	case "f4", "ctrl+d":
		if m.inventory != nil {
			return m, func() tea.Msg { return DeployRequestedMsg{} }
		}
	case "f5":
		if n := m.cloneableSelection(); n != nil && !m.cloning {
			m.cloneActive = true
			m.cloneErr = nil
			m.cloneRef = n.Ref
			m.cloneSource = n.Name
			m.cloneInput.SetValue(n.Name + "-clone")
			m.cloneInput.CursorEnd()
			m.cloneInput.Focus()
			return m, textinput.Blink
		}
	case "f6":
		if n := m.cloneableSelection(); n != nil {
			return m, func() tea.Msg {
				return ExportRequestedMsg{Ref: n.Ref, Name: n.Name, Datacenter: m.tree.SelectedDatacenter()}
			}
		}
	case "f7":
		return m, func() tea.Msg { return SettingsRequestedMsg{} }
	case "f8":
		return m, func() tea.Msg { return LogsRequestedMsg{} }
	case "f9":
		return m, func() tea.Msg { return MenuRequestedMsg{} }
	case "f10":
		return m, func() tea.Msg { return QuitRequestedMsg{} }
	}
	return m, nil
}

func (m *Model) updateCloneModal(keyMsg tea.KeyMsg) (*Model, tea.Cmd) {
	switch keyMsg.String() {
	case "esc":
		m.cloneActive = false
		return m, nil
	case "enter":
		name := m.cloneInput.Value()
		if name == "" {
			m.cloneErr = fmt.Errorf("enter a name for the clone")
			return m, nil
		}
		ref := m.cloneRef
		m.cloneActive = false
		m.cloning = true
		return m, func() tea.Msg { return CloneRequestedMsg{Ref: ref, NewName: name} }
	}
	var cmd tea.Cmd
	m.cloneInput, cmd = m.cloneInput.Update(keyMsg)
	return m, cmd
}

func (m *Model) keyHints() []components.KeyHint {
	canActOnSelection := m.cloneableSelection() != nil
	return []components.KeyHint{
		{Key: "F1", Label: "Help", Enabled: true},
		{Key: "F2", Label: "Connect", Enabled: true},
		{Key: "F3", Label: "Refresh", Enabled: true},
		{Key: "F4", Label: "Deploy", Enabled: m.inventory != nil},
		{Key: "F5", Label: "Clone", Enabled: canActOnSelection && !m.cloning},
		{Key: "F6", Label: "Export", Enabled: canActOnSelection},
		{Key: "F7", Label: "Settings", Enabled: true},
		{Key: "F8", Label: "Logs", Enabled: true},
		{Key: "F9", Label: "Menu", Enabled: true},
		{Key: "F10", Label: "Quit", Enabled: true},
	}
}

func (m *Model) renderInfoBar() string {
	if m.inventory == nil {
		return components.RenderInfoBar(m.styles, false, "", nil)
	}
	s := m.inventory.Summary

	metrics := []components.Metric{
		{Label: "🏠 Host", Value: s.Host},
		{Label: "🔖 Version", Value: s.Version},
		{Label: "⚙ CPU", Value: strconv.Itoa(int(s.CPUCores)) + " Cores"},
		{Label: "🧠 Memory", Value: formatGB(s.MemoryBytes)},
		{Label: "🖥 VMs", Value: strconv.Itoa(s.VMCount)},
		{Label: "💽 Datastores", Value: strconv.Itoa(s.DatastoreCount)},
		{Label: "🌐 Networks", Value: strconv.Itoa(s.NetworkCount)},
		{Label: "📄 Templates", Value: strconv.Itoa(s.TemplateCount)},
	}

	label := m.connectionLabel
	if m.refreshing {
		label += " (refreshing…)"
	}
	if m.cloning {
		label += " (cloning…)"
	}
	return components.RenderInfoBar(m.styles, true, label, metrics)
}

// usageBarRowCount returns how many rows renderUsageBars will produce, so
// SetSize can reserve the right amount of vertical space for it.
func (m *Model) usageBarRowCount() int {
	if m.inventory == nil {
		return 0
	}
	return 2 + len(m.inventory.Summary.Datastores)
}

func (m *Model) renderUsageBars() string {
	if m.inventory == nil {
		return ""
	}
	s := m.inventory.Summary

	bars := []components.UsageBar{
		{
			Icon: "⚙", Label: "CPU", Used: float64(s.CPUUsageMHz), Total: float64(s.CPUTotalMHz),
			Format: func(used, total float64) string {
				return fmt.Sprintf("%.1f/%.1f GHz", used/1000, total/1000)
			},
		},
		{
			Icon: "🧠", Label: "RAM", Used: float64(s.MemoryUsageBytes), Total: float64(s.MemoryBytes),
			Format: func(used, total float64) string {
				return fmt.Sprintf("%.0f/%.0f GB", used/(1<<30), total/(1<<30))
			},
		},
	}
	for _, ds := range s.Datastores {
		usedBytes := float64(ds.CapacityBytes - ds.FreeBytes)
		bars = append(bars, components.UsageBar{
			Icon: "💽", Label: ds.Name, Used: usedBytes, Total: float64(ds.CapacityBytes),
			Format: func(used, total float64) string {
				return fmt.Sprintf("%.0f/%.0f GB", used/(1<<30), total/(1<<30))
			},
		})
	}

	return components.RenderUsageBars(m.styles, m.width, bars)
}

func (m *Model) renderCloneModal() string {
	title := m.styles.PanelTitle.Render("🧬 Clone \"" + m.cloneSource + "\"")
	body := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		m.styles.InputLabel.Render("New VM name"),
		m.styles.InputFocus.Render(m.cloneInput.View()),
	)
	if m.cloneErr != nil {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", m.styles.Error.Render("⚠ "+m.cloneErr.Error()))
	}
	box := m.styles.PanelFocused.Width(60).Render(body)
	return lipgloss.Place(m.width, m.bodyHeight, lipgloss.Center, lipgloss.Center, box)
}

// View renders the full dashboard screen.
func (m *Model) View() string {
	titleBar := components.RenderTitleBar(m.styles, m.width, "🖥", "", components.RenderTitleBarCredit(m.styles))
	infoBar := m.renderInfoBar()
	usageBar := m.renderUsageBars()

	var body string
	if m.cloneActive {
		body = m.renderCloneModal()
	} else {
		left := m.styles.PanelFocused.
			Width(m.leftWidth - 2).
			Height(m.bodyHeight - 2).
			Render(m.tree.View())

		details := components.RenderDetails(m.styles, m.tree.Selected())
		if m.err != nil {
			details = m.styles.Error.Render("⚠ " + m.err.Error())
		}
		right := m.styles.PanelBlurred.
			Width(m.rightWidth - 2).
			Height(m.bodyHeight - 2).
			Render(details)

		body = lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	}

	statusBar := components.RenderStatusBar(m.styles, m.width, m.keyHints())

	rows := []string{titleBar, infoBar}
	if usageBar != "" {
		rows = append(rows, usageBar)
	}
	rows = append(rows, body, statusBar)
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func formatGB(b int64) string {
	gb := float64(b) / (1024 * 1024 * 1024)
	return strconv.Itoa(int(gb+0.5)) + " GB"
}
