package deploywizard

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/deploy"
)

type deployEventMsg deploy.Event
type deployChannelClosedMsg struct{}
type progressTickMsg struct{}

func tickProgress() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return progressTickMsg{} })
}

func waitForDeployEvent(events chan deploy.Event) tea.Cmd {
	return func() tea.Msg {
		e, ok := <-events
		if !ok {
			return deployChannelClosedMsg{}
		}
		return deployEventMsg(e)
	}
}

// startDeploy launches the ovftool command built from the wizard's spec
// and switches to the live progress step. It's called both for the
// initial deploy (from the summary step) and for retries.
func (m *Model) startDeploy() (*Model, tea.Cmd) {
	ovftoolPath, err := deploy.LocateOVFTool()
	if err != nil {
		m.err = err
		return m, nil
	}

	m.step = stepProgress
	m.running = true
	m.done = false
	m.succeeded = false
	m.percent = 0
	m.lastLine = "Starting…"
	m.logLines = nil
	m.log.SetContent("")
	m.deployErr = nil
	m.startTime = time.Now()

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	events := make(chan deploy.Event, 256)
	m.events = events
	args := m.args

	go func() {
		defer close(events)
		_ = deploy.Run(ctx, ovftoolPath, args, func(e deploy.Event) {
			events <- e
		})
	}()

	return m, tea.Batch(waitForDeployEvent(events), tickProgress())
}

// CancelActiveDeploy stops an in-flight deploy, if one is running. The
// parent app calls this on quit/disconnect so an ovftool child process
// never keeps running detached from the UI that started it.
func (m *Model) CancelActiveDeploy() {
	if m.running && m.cancel != nil {
		m.cancel()
	}
}

func (m *Model) updateProgress(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case deployEventMsg:
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
			m.deployErr = e.Err
			m.elapsed = time.Since(m.startTime).Round(time.Second)
			if e.Success {
				m.percent = 100
			}
		}
		return m, waitForDeployEvent(m.events)

	case deployChannelClosedMsg:
		return m, nil

	case progressTickMsg:
		if m.running {
			return m, tickProgress()
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
				m.step = stepSummary
			}
		case "r":
			if m.done && !m.succeeded {
				return m.startDeploy()
			}
		}
	}
	return m, nil
}

func (m *Model) viewProgress() string {
	elapsed := m.elapsed
	if !m.done {
		elapsed = time.Since(m.startTime).Round(time.Second)
	}

	status := m.styles.TreeMuted.Render("Running…")
	if m.done {
		if m.succeeded {
			status = m.styles.Success.Render("✓ Completed successfully")
		} else {
			status = m.styles.Error.Render("✗ Deployment failed")
		}
	}

	barWidth := 40
	filled := m.percent * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}
	bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled) + "]"

	header := lipgloss.JoinVertical(lipgloss.Left,
		m.styles.PanelTitle.Render("Deploying "+m.spec.VMName),
		fmt.Sprintf("%s %d%%   ⏱ %s", bar, m.percent, elapsed),
		status,
		m.styles.TreeMuted.Render(m.lastLine),
		"",
	)

	logBox := m.styles.PanelBlurred.Width(m.width - 2).Height(m.log.Height + 2).Render(m.log.View())

	help := "Esc cancel"
	if m.done && m.succeeded {
		help = "Enter/Esc back to dashboard"
	} else if m.done && !m.succeeded {
		help = "r retry   Esc back to summary"
	}

	panel := m.styles.PanelFocused.Width(m.width - 2).Render(header)
	return lipgloss.JoinVertical(lipgloss.Left,
		m.titleBar(), m.stepper(), panel, logBox,
		m.styles.StatusBar.Render(help),
	)
}
