package deploywizard

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/deploy"
)

// buildSpec assembles a deploy.Spec (and its ovftool argv) from every
// choice collected across steps 1-5. Called on entering the summary step
// and again if the user backs up and changes something.
func (m *Model) buildSpec() {
	spec := deploy.Spec{
		SourcePath:       m.sourcePath,
		VMName:           m.vmNameInput.Value(),
		Hostname:         m.profile.Hostname,
		Port:             m.profile.Port,
		Username:         m.profile.Username,
		Password:         m.password,
		IgnoreSSL:        m.profile.IgnoreSSL,
		DiskMode:         m.diskMode(),
		PowerOn:          m.powerOn,
		ImportAsTemplate: m.importAsTemplate,
		Properties:       map[string]string{},
	}

	if !m.isESXi && len(m.computeOptions) > 0 {
		c := m.computeOptions[m.computeCursor]
		spec.DatacenterName = c.DatacenterName
		spec.ComputeResourceName = c.Name
	}
	if len(m.datastoreOptions) > 0 {
		spec.DatastoreRef = m.datastoreOptions[m.datastoreCursor].Ref
	}
	for i, n := range m.descriptor.Networks {
		if idx := m.networkMap[i]; idx >= 0 && idx < len(m.networkTargets) {
			spec.Networks = append(spec.Networks, deploy.NetworkMapping{
				OVFName: n.Name, TargetRef: m.networkTargets[idx].Ref,
			})
		}
	}
	for i, p := range m.descriptor.Properties {
		if i < len(m.propertyInputs) {
			if v := m.propertyInputs[i].Value(); v != "" {
				spec.Properties[p.Key] = v
			}
		}
	}
	spec.ExtraArgs = m.extraArgsInput.Value()

	m.spec = spec
	m.args = deploy.BuildArgs(spec)

	ovftoolPath, err := deploy.LocateOVFTool()
	if err != nil {
		m.command = "ovftool " + err.Error()
		return
	}
	m.command = deploy.RedactedCommandLine(ovftoolPath, m.args)
}

func (m *Model) updateSummary(msg tea.Msg) (*Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		m.step = stepAdvanced
		return m, nil
	case "s":
		return m, m.saveDeploymentProfile()
	case "enter", "d":
		return m.startDeploy()
	}
	return m, nil
}

func (m *Model) saveDeploymentProfile() tea.Cmd {
	profile := profileFromSpec(m.vmNameInput.Value()+" profile", m.spec)
	_, _ = m.cfg.SaveDeploymentProfile(profile)
	return nil
}

func (m *Model) viewSummary() string {
	title := m.styles.PanelTitle.Render("Summary")

	rows := []string{
		m.styles.DetailLabel.Render("Source") + m.styles.DetailValue.Render(m.sourcePath),
		m.styles.DetailLabel.Render("VM Name") + m.styles.DetailValue.Render(m.spec.VMName),
		m.styles.DetailLabel.Render("Disk Mode") + m.styles.DetailValue.Render(string(m.spec.DiskMode)),
		"",
		m.styles.PanelTitle.Render("Generated ovftool command"),
		m.styles.TreeMuted.Render(wrapText(m.command, m.width-6)),
	}

	if _, err := deploy.LocateOVFTool(); err != nil {
		rows = append(rows, "", m.styles.Error.Render("⚠ "+err.Error()))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, append([]string{title, ""}, rows...)...)
	panel := m.styles.PanelFocused.Width(m.width - 2).Render(body)
	help := m.styles.StatusBar.Render("Enter/d deploy   s save as profile   Esc back")

	return lipgloss.JoinVertical(lipgloss.Left, m.titleBar(), m.stepper(), panel, help)
}

func wrapText(s string, width int) string {
	if width < 10 {
		width = 10
	}
	var out []string
	line := ""
	for _, word := range strings.Fields(s) {
		if len(line)+len(word)+1 > width {
			out = append(out, line)
			line = word
			continue
		}
		if line == "" {
			line = word
		} else {
			line += " " + word
		}
	}
	if line != "" {
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
