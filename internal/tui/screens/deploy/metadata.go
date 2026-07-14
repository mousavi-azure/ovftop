package deploywizard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/deploy"
	"github.com/mousavi-azure/ovftop/internal/deploy/ovf"
)

func (m *Model) startProbe() tea.Cmd {
	m.probing = true
	m.err = nil
	source := m.sourcePath
	return func() tea.Msg {
		ovftoolPath, err := deploy.LocateOVFTool()
		if err != nil {
			return probeDoneMsg{err: err}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		d, err := ovf.Probe(ctx, ovftoolPath, source)
		return probeDoneMsg{descriptor: d, err: err}
	}
}

func (m *Model) updateMetadata(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case probeDoneMsg:
		m.probing = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.descriptor = msg.descriptor
		m.initFromDescriptor()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "left":
			m.step = stepSource
			m.descriptor = nil
			m.err = nil
			return m, nil
		case "r":
			if m.err != nil {
				return m, m.startProbe()
			}
		case "enter", "right":
			if m.descriptor != nil && !m.probing {
				m.step = stepSettings
			}
		}
	}
	return m, nil
}

// initFromDescriptor seeds the later steps' default state once metadata
// is available: a sanitized default VM name, best-effort network
// auto-matching, and one text input per OVF property.
func (m *Model) initFromDescriptor() {
	name := m.descriptor.Name
	if name == "" && len(m.descriptor.VMs) > 0 {
		name = m.descriptor.VMs[0].Name
	}
	if name == "" {
		name = "new-vm"
	}
	m.vmNameInput.SetValue(sanitizeVMName(name))

	m.networkMap = make([]int, len(m.descriptor.Networks))
	for i, n := range m.descriptor.Networks {
		m.networkMap[i] = -1
		for j, t := range m.networkTargets {
			if strings.EqualFold(t.Name, n.Name) {
				m.networkMap[i] = j
				break
			}
		}
	}

	m.propertyInputs = make([]textinput.Model, len(m.descriptor.Properties))
	for i, p := range m.descriptor.Properties {
		m.propertyInputs[i] = newTextInput(p.Default, false)
	}
}

func sanitizeVMName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	s := b.String()
	if s == "" {
		return "new-vm"
	}
	return s
}

func (m *Model) viewMetadata() string {
	title := m.styles.PanelTitle.Render("📦 " + m.sourcePath)

	var body string
	switch {
	case m.probing:
		body = m.styles.TreeMuted.Render("Reading OVF/OVA metadata…")
	case m.err != nil:
		body = m.styles.Error.Render("⚠ " + m.err.Error())
	case m.descriptor != nil:
		body = m.renderDescriptor(m.descriptor)
	}

	help := "Enter continue   Esc back"
	if m.err != nil {
		help = "r retry   Esc back"
	}

	panel := m.styles.PanelFocused.Width(m.width - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", body),
	)
	return lipgloss.JoinVertical(lipgloss.Left,
		m.titleBar(), m.stepper(), panel,
		m.styles.StatusBar.Render(help),
	)
}

func (m *Model) renderDescriptor(d *ovf.Descriptor) string {
	label := m.styles.DetailLabel
	value := m.styles.DetailValue
	row := func(l, v string) string {
		if v == "" {
			return ""
		}
		return label.Render(l) + value.Render(v)
	}

	lines := []string{
		row("🏷 Name", d.Name),
		row("🏭 Vendor", d.Vendor),
		row("🔖 Version", d.FullVersion),
		row("📝 Description", d.Description),
	}

	if len(d.VMs) > 0 {
		vm := d.VMs[0]
		lines = append(lines,
			"",
			row("⚙ CPUs", fmt.Sprintf("%d", vm.NumCPUs)),
			row("🧠 Memory", formatBytes(vm.MemoryBytes)),
			row("🐧 Guest OS", vm.OSID),
		)
		for i, disk := range vm.Disks {
			lines = append(lines, row(fmt.Sprintf("💾 Disk %d", i+1), formatBytes(disk.CapacityBytes)))
		}
	}

	if len(d.Networks) > 0 {
		lines = append(lines, "")
		var names []string
		for _, n := range d.Networks {
			names = append(names, n.Name)
		}
		lines = append(lines, row("🌐 Networks", strings.Join(names, ", ")))
	}

	if len(d.Properties) > 0 {
		lines = append(lines, "", m.styles.PanelTitle.Render("OVF Properties"))
		for _, p := range d.Properties {
			desc := p.Description
			if desc == "" {
				desc = p.Label
			}
			lines = append(lines, row("• "+p.Key, desc))
		}
	}

	if len(d.Warnings) > 0 {
		lines = append(lines, "", m.styles.Warning.Render("⚠ Warnings"))
		for _, w := range d.Warnings {
			lines = append(lines, m.styles.Warning.Render("  • "+w))
		}
	}

	var out []string
	for _, l := range lines {
		if l != "" {
			out = append(out, l)
		} else {
			out = append(out, "")
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, out...)
}
