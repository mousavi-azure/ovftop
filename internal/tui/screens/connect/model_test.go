package connect

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/mousavi-azure/ovftop/internal/config"
	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

func newTestManager(t *testing.T) *config.Manager {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	return m
}

func typeText(t *testing.T, m *Model, s string) *Model {
	t.Helper()
	for _, r := range s {
		var msg tea.KeyMsg
		if r == ' ' {
			msg = tea.KeyMsg{Type: tea.KeySpace}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
		}
		m, _ = m.Update(msg)
	}
	return m
}

func TestNewConnectionRequiresRequiredFields(t *testing.T) {
	m := New(theme.New(theme.Dark), newTestManager(t))
	m.SetSize(80, 24)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.mode != modeForm {
		t.Fatalf("expected modeForm after 'n', got %v", m.mode)
	}

	// Submit immediately with everything blank.
	for m.focus != fieldSubmit {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.err == nil {
		t.Fatal("expected validation error for blank required fields")
	}
	if m.mode != modeForm {
		t.Fatal("expected to remain in form mode after validation failure")
	}
}

func TestNewConnectionSubmitEmitsConnectRequested(t *testing.T) {
	mgr := newTestManager(t)
	m := New(theme.New(theme.Dark), mgr)
	m.SetSize(80, 24)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	m = typeText(t, m, "Lab ESXi")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // -> description
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // -> type toggle
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // -> hostname
	m = typeText(t, m, "esxi01.lab.local")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // -> username
	m = typeText(t, m, "root")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // -> password
	m = typeText(t, m, "hunter2")

	for m.focus != fieldSubmit {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}

	var cmd tea.Cmd
	m, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.err != nil {
		t.Fatalf("unexpected validation error: %v", m.err)
	}
	if cmd == nil {
		t.Fatal("expected a command emitting ConnectRequestedMsg")
	}

	msg := cmd()
	req, ok := msg.(ConnectRequestedMsg)
	if !ok {
		t.Fatalf("expected ConnectRequestedMsg, got %T", msg)
	}
	if req.Profile.Hostname != "esxi01.lab.local" || req.Profile.Username != "root" {
		t.Errorf("unexpected profile: %+v", req.Profile)
	}
	if req.Password != "hunter2" {
		t.Errorf("expected password hunter2, got %q", req.Password)
	}

	// The profile should now be persisted (Remember defaults to unchecked,
	// so no vaulted password is expected, but the profile record itself
	// should exist for next time).
	if got := len(mgr.Profiles()); got != 1 {
		t.Fatalf("expected 1 saved profile, got %d", got)
	}
}

func TestEscCancelsFormWithoutSaving(t *testing.T) {
	mgr := newTestManager(t)
	m := New(theme.New(theme.Dark), mgr)
	m.SetSize(80, 24)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = typeText(t, m, "Lab ESXi")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m.mode != modeList {
		t.Fatalf("expected modeList after Esc, got %v", m.mode)
	}
	if got := len(mgr.Profiles()); got != 0 {
		t.Fatalf("expected no persisted profile after cancel, got %d", got)
	}
}

func TestConnectionTypeToggle(t *testing.T) {
	m := New(theme.New(theme.Dark), newTestManager(t))
	m.SetSize(80, 24)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // fieldName -> fieldDescription
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab}) // fieldDescription -> fieldType

	if m.connType != config.ConnectionESXi {
		t.Fatalf("expected default type ESXi, got %v", m.connType)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.connType != config.ConnectionVCenter {
		t.Fatalf("expected vCenter after toggle, got %v", m.connType)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.connType != config.ConnectionESXi {
		t.Fatalf("expected ESXi after toggling back, got %v", m.connType)
	}
}
