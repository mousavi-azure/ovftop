// Package export implements the F6 Export screen: pick a local destination
// folder, then export the selected VM/template to an OVA file there via
// ovftool (the mirror image of the Deploy wizard's import flow).
package export

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/deploy"
	"github.com/mousavi-azure/ovftop/internal/tui/components"
	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

type step int

const (
	stepBrowse step = iota
	stepProgress
)

// ClosedMsg is emitted when the user backs out of (or finishes) the
// export screen.
type ClosedMsg struct{}

// Params describes the connection and the VM/template being exported.
type Params struct {
	Hostname   string
	Port       int
	Username   string
	Password   string
	Datacenter string // "" when connected directly to a standalone ESXi host
	VMName     string
}

// sourceLocator builds the vi:// locator ovftool expects for an export
// source: a VM's own path within the connected endpoint, as opposed to
// the compute-resource locator used when deploying (see
// internal/deploy.TargetLocator).
func sourceLocator(p Params) string {
	userinfo := p.Username + ":" + p.Password
	host := p.Hostname
	if p.Port != 0 {
		host = fmt.Sprintf("%s:%d", p.Hostname, p.Port)
	}
	path := p.VMName
	if p.Datacenter != "" {
		path = p.Datacenter + "/vm/" + p.VMName
	}
	return fmt.Sprintf("vi://%s@%s/%s", userinfo, host, path)
}

type eventMsg deploy.Event
type channelClosedMsg struct{}
type tickMsg struct{}

// Model is the export screen.
type Model struct {
	styles  theme.Styles
	params  Params
	browser *components.FileBrowser

	step step

	running   bool
	done      bool
	succeeded bool
	percent   int
	lastLine  string
	logLines  []string
	log       viewport.Model
	err       error

	startTime time.Time
	elapsed   time.Duration
	cancel    context.CancelFunc
	events    chan deploy.Event

	width, height int
}

// New creates the export screen, starting the destination browser in the
// user's home directory.
func New(styles theme.Styles, params Params) *Model {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	b := components.NewFileBrowser(styles, home)
	b.ToggleShowAll() // picking a destination folder, not an OVA/OVF source
	return &Model{
		styles:  styles,
		params:  params,
		browser: b,
		log:     viewport.New(80, 10),
	}
}

// SetSize updates the viewport dimensions.
func (m *Model) SetSize(w, h int) {
	m.width, m.height = w, h
	m.browser.SetSize(w-4, h-10)
	m.log.Width = w - 4
	m.log.Height = 8
}

// Init satisfies the tea.Model-shaped contract used by the parent app.
func (m *Model) Init() tea.Cmd { return nil }

// CancelActiveExport stops an in-flight export, if one is running. The
// parent app calls this on quit/disconnect so an ovftool child process
// never keeps running detached from the UI that started it.
func (m *Model) CancelActiveExport() {
	if m.running && m.cancel != nil {
		m.cancel()
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func waitForEvent(events chan deploy.Event) tea.Cmd {
	return func() tea.Msg {
		e, ok := <-events
		if !ok {
			return channelClosedMsg{}
		}
		return eventMsg(e)
	}
}

func (m *Model) startExport(destDir string) (*Model, tea.Cmd) {
	ovftoolPath, err := deploy.LocateOVFTool()
	if err != nil {
		m.err = err
		return m, nil
	}

	args := []string{
		"--noSSLVerify",
		"--acceptAllEulas",
		"--overwrite",
		sourceLocator(m.params),
		destDir + "/" + m.params.VMName + ".ova",
	}

	m.step = stepProgress
	m.running = true
	m.done = false
	m.succeeded = false
	m.percent = 0
	m.lastLine = "Starting…"
	m.logLines = nil
	m.log.SetContent("")
	m.err = nil
	m.startTime = time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	events := make(chan deploy.Event, 256)
	m.events = events

	go func() {
		defer close(events)
		_ = deploy.Run(ctx, ovftoolPath, args, func(e deploy.Event) {
			events <- e
		})
	}()

	return m, tea.Batch(waitForEvent(events), tick())
}

// Update handles a message and returns the (possibly replaced) model plus
// any command, mirroring tea.Model.Update for use as a sub-model.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	if m.step == stepProgress {
		return m.updateProgress(msg)
	}
	return m.updateBrowse(msg)
}

func (m *Model) updateBrowse(msg tea.Msg) (*Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "esc":
		return m, func() tea.Msg { return ClosedMsg{} }
	case "up", "k":
		m.browser.MoveUp()
	case "down", "j":
		m.browser.MoveDown()
	case "left", "h", "backspace":
		m.browser.GoUp()
	case "right", "l", "enter":
		if sel := m.browser.Selected(); sel != nil && sel.IsDir {
			m.browser.Open()
		}
	case ".":
		m.browser.ToggleHidden()
	case "s":
		return m.startExport(m.browser.CurrentDir())
	}
	return m, nil
}

func (m *Model) updateProgress(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case eventMsg:
		e := deploy.Event(msg)
		if e.Kind == deploy.EventLine {
			m.lastLine = e.Text
			if e.Percent >= 0 {
				m.percent = e.Percent
			}
			m.logLines = append(m.logLines, e.Text)
			m.log.SetContent(strings.Join(m.logLines, "\n"))
			m.log.GotoBottom()
		} else {
			m.running = false
			m.done = true
			m.succeeded = e.Success
			m.err = e.Err
			m.elapsed = time.Since(m.startTime).Round(time.Second)
			if e.Success {
				m.percent = 100
			}
		}
		return m, waitForEvent(m.events)

	case channelClosedMsg:
		return m, nil

	case tickMsg:
		if m.running {
			return m, tick()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "c", "enter":
			if m.running && m.cancel != nil {
				m.cancel()
				return m, nil
			}
			if m.done {
				if m.succeeded {
					return m, func() tea.Msg { return ClosedMsg{} }
				}
				m.step = stepBrowse
			}
		case "r":
			if m.done && !m.succeeded {
				return m.startExport(m.browser.CurrentDir())
			}
		}
	}
	return m, nil
}

func (m *Model) titleBar() string {
	return components.RenderTitleBar(m.styles, m.width, "📤", "Export \""+m.params.VMName+"\"", "")
}

// View renders the export screen.
func (m *Model) View() string {
	if m.step == stepProgress {
		return m.viewProgress()
	}
	return m.viewBrowse()
}

func (m *Model) viewBrowse() string {
	title := m.styles.PanelTitle.Render("Choose a destination folder for \"" + m.params.VMName + ".ova\"")
	body := lipgloss.JoinVertical(lipgloss.Left, title, "", m.browser.View())
	if m.err != nil {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", m.styles.Error.Render("⚠ "+m.err.Error()))
	}
	panel := m.styles.PanelFocused.Width(m.width - 2).Render(body)
	help := "↑/↓ move   → open dir   ← up dir   . hidden   s export here   Esc cancel"
	return lipgloss.JoinVertical(lipgloss.Left, m.titleBar(), panel, m.styles.StatusBar.Render(help))
}

func (m *Model) viewProgress() string {
	elapsed := m.elapsed
	if !m.done {
		elapsed = time.Since(m.startTime).Round(time.Second)
	}

	status := m.styles.TreeMuted.Render("Running…")
	if m.done {
		if m.succeeded {
			status = m.styles.Success.Render("✓ Export completed successfully")
		} else {
			status = m.styles.Error.Render("✗ Export failed")
		}
	}

	barWidth := 40
	filled := m.percent * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}
	bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled) + "]"

	header := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.PanelTitle.Render("Exporting "+m.params.VMName),
		fmt.Sprintf("%s %d%%   ⏱ %s", bar, m.percent, elapsed),
		status,
		m.styles.TreeMuted.Render(m.lastLine),
		"",
	)
	if m.err != nil && m.done && !m.succeeded {
		header = lipgloss.JoinVertical(lipgloss.Left, header, m.styles.Error.Render("⚠ "+m.err.Error()))
	}

	logBox := m.styles.PanelBlurred.Width(m.width - 2).Height(m.log.Height + 2).Render(m.log.View())

	help := "Esc cancel"
	if m.done && m.succeeded {
		help = "Enter/Esc back to dashboard"
	} else if m.done && !m.succeeded {
		help = "r retry   Esc back to folder picker"
	}

	panel := m.styles.PanelFocused.Width(m.width - 2).Render(header)
	return lipgloss.JoinVertical(lipgloss.Left,
		m.titleBar(), panel, logBox,
		m.styles.StatusBar.Render(help),
	)
}
