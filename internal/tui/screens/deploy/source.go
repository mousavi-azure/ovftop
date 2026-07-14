package deploywizard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/deploy"
	"github.com/mousavi-azure/ovftop/internal/deploy/ovf"
)

// downloadState is written from the Download goroutine's progress
// callback and read from a periodic tea.Tick command, since a single
// tea.Cmd invocation can't stream multiple messages back on its own.
type downloadState struct {
	mu   sync.Mutex
	prog deploy.DownloadProgress
}

func (s *downloadState) set(p deploy.DownloadProgress) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prog = p
}

func (s *downloadState) get() deploy.DownloadProgress {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.prog
}

type downloadTickMsg struct{}
type downloadDoneMsg struct {
	path string
	err  error
}
type probeDoneMsg struct {
	descriptor *ovf.Descriptor
	err        error
}

func downloadDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "ovftop", "downloads")
}

func (m *Model) startDownload(url string) tea.Cmd {
	m.downloading = true
	m.err = nil
	state := &downloadState{}
	m.downloadState = state

	download := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		path, err := deploy.Download(ctx, url, downloadDir(), state.set)
		return downloadDoneMsg{path: path, err: err}
	}
	return tea.Batch(download, tickDownloadProgress())
}

func tickDownloadProgress() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return downloadTickMsg{} })
}

func (m *Model) updateSource(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case downloadTickMsg:
		if !m.downloading {
			return m, nil
		}
		m.downloadInfo = m.downloadState.get()
		return m, tickDownloadProgress()

	case downloadDoneMsg:
		m.downloading = false
		if msg.err != nil {
			m.err = fmt.Errorf("download failed: %w", msg.err)
			return m, nil
		}
		m.sourcePath = msg.path
		m.step = stepMetadata
		return m, m.startProbe()

	case tea.KeyMsg:
		return m.updateSourceKey(msg)
	}
	return m, nil
}

func (m *Model) updateSourceKey(msg tea.KeyMsg) (*Model, tea.Cmd) {
	if m.downloading {
		return m, nil
	}

	switch msg.String() {
	case "tab":
		if m.sourceMethod == methodURL {
			m.sourceMethod = methodBrowse
			m.urlInput.Blur()
		} else {
			m.sourceMethod = methodURL
			m.urlInput.Focus()
		}
		return m, nil
	}

	if m.sourceMethod == methodURL {
		switch msg.String() {
		case "enter":
			url := strings.TrimSpace(m.urlInput.Value())
			if url == "" {
				m.err = fmt.Errorf("enter a URL first")
				return m, nil
			}
			return m, m.startDownload(url)
		}
		var cmd tea.Cmd
		m.urlInput, cmd = m.urlInput.Update(msg)
		return m, cmd
	}

	// Browse mode.
	switch msg.String() {
	case "up", "k":
		m.browser.MoveUp()
	case "down", "j":
		m.browser.MoveDown()
	case "left", "h", "backspace":
		m.browser.GoUp()
	case "right", "l", "enter":
		sel := m.browser.Selected()
		if sel == nil {
			return m, nil
		}
		if sel.IsDir {
			m.browser.Open()
			return m, nil
		}
		m.sourcePath = m.browser.SelectedPath()
		m.step = stepMetadata
		return m, m.startProbe()
	case ".":
		m.browser.ToggleHidden()
	case "a":
		m.browser.ToggleShowAll()
	}
	return m, nil
}

func (m *Model) viewSource() string {
	title := m.styles.PanelTitle.Render("Select an OVF/OVA source")

	methodRow := func(label string, active bool) string {
		if active {
			return m.styles.KeyHint.Render(" " + label + " ")
		}
		return m.styles.KeyHintOff.Render(" " + label + " ")
	}
	methods := lipgloss.JoinHorizontal(lipgloss.Top,
		methodRow("① Download from URL", m.sourceMethod == methodURL),
		" ",
		methodRow("② Browse Local Files", m.sourceMethod == methodBrowse),
	)

	var body string
	switch {
	case m.downloading:
		body = m.renderDownloadProgress()
	case m.sourceMethod == methodURL:
		box := m.styles.InputFocus
		if m.sourceMethod != methodURL {
			box = m.styles.InputBox
		}
		body = lipgloss.JoinVertical(lipgloss.Left,
			m.styles.InputLabel.Render("🌐 URL"),
			box.Render(m.urlInput.View()),
			"",
			m.styles.TreeMuted.Render("Downloaded files are cached under "+downloadDir()),
		)
	default:
		body = m.browser.View()
	}

	if m.err != nil {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", m.styles.Error.Render("⚠ "+m.err.Error()))
	}

	help := "Tab switch method   Enter select/download   Esc cancel"
	if m.sourceMethod == methodBrowse {
		help = "↑/↓ move   → open/select   ← up dir   . hidden   a show all   Tab switch method   Esc cancel"
	}

	panel := m.styles.PanelFocused.Width(m.width - 2).Render(
		lipgloss.JoinVertical(lipgloss.Left, title, "", methods, "", body),
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		m.titleBar(), m.stepper(), panel,
		m.styles.StatusBar.Render(help),
	)
}

func (m *Model) renderDownloadProgress() string {
	p := m.downloadInfo
	width := 40
	pct := 0.0
	if p.TotalBytes > 0 {
		pct = float64(p.BytesRead) / float64(p.TotalBytes)
	}
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}
	bar := "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"

	speed := ""
	if p.Elapsed > 0 {
		bps := float64(p.BytesRead) / p.Elapsed.Seconds()
		speed = fmt.Sprintf("%.1f MB/s", bps/(1024*1024))
	}

	sizeText := fmt.Sprintf("%s", formatBytes(p.BytesRead))
	if p.TotalBytes > 0 {
		sizeText += " / " + formatBytes(p.TotalBytes)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.styles.PanelTitle.Render("Downloading…"),
		bar,
		fmt.Sprintf("%s   %s", sizeText, speed),
	)
}

func formatBytes(n int64) string {
	units := []string{"B", "KB", "MB", "GB"}
	f := float64(n)
	i := 0
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d %s", n, units[i])
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}
