package theme

import "github.com/charmbracelet/lipgloss"

// Styles is the full set of lipgloss styles derived from a Palette. Every
// screen and component renders through these rather than building ad hoc
// styles, so switching palettes re-themes the whole app.
type Styles struct {
	Palette Palette

	App lipgloss.Style

	TitleBar        lipgloss.Style
	TitleBarBrand   lipgloss.Style
	TitleBarSection lipgloss.Style
	StatusBar       lipgloss.Style
	KeyHint         lipgloss.Style
	KeyHintOff      lipgloss.Style
	KeyBadge        lipgloss.Style
	KeyBadgeOff     lipgloss.Style
	KeyLabel        lipgloss.Style
	KeyLabelOff     lipgloss.Style

	PanelFocused lipgloss.Style
	PanelBlurred lipgloss.Style
	PanelTitle   lipgloss.Style

	TreeGroup    lipgloss.Style
	TreeItem     lipgloss.Style
	TreeSelected lipgloss.Style
	TreeMuted    lipgloss.Style

	DetailLabel lipgloss.Style
	DetailValue lipgloss.Style

	MetricTile      lipgloss.Style
	MetricLabel     lipgloss.Style
	MetricValue     lipgloss.Style
	ConnectedBadge  lipgloss.Style
	DisconnectedBad lipgloss.Style

	Error   lipgloss.Style
	Success lipgloss.Style
	Warning lipgloss.Style

	InputLabel lipgloss.Style
	InputBox   lipgloss.Style
	InputFocus lipgloss.Style
}

// New derives a full Styles set from a Palette.
func New(p Palette) Styles {
	s := Styles{Palette: p}

	s.App = lipgloss.NewStyle().Foreground(p.Foreground)

	// The title bar carries two tones on one solid background: a bold
	// brand segment ("OVFTOP") and a lighter-weight section segment
	// (the current screen name), so the product name always reads as the
	// visual anchor instead of competing evenly with the screen label.
	s.TitleBar = lipgloss.NewStyle().
		Background(p.Primary).
		Padding(0, 1)
	s.TitleBarBrand = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.HighlightFg).
		Background(p.Primary)
	s.TitleBarSection = lipgloss.NewStyle().
		Foreground(p.HighlightFg).
		Background(p.Primary)

	s.StatusBar = lipgloss.NewStyle().
		Foreground(p.Foreground).
		Background(p.BorderBlur)

	// Enabled F-key hints render as little green pill buttons (rounded
	// caps + filled body); disabled ones get the same pill shape in a
	// muted tone so the footer reads as one consistent row of buttons,
	// with color alone telling you what's actually clickable right now.
	const buttonDark = lipgloss.Color("235")
	s.KeyHint = lipgloss.NewStyle().
		Bold(true).
		Foreground(buttonDark).
		Background(p.Success).
		Padding(0, 1)

	s.KeyHintOff = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Foreground).
		Background(p.Dim).
		Padding(0, 1)

	// The bottom status bar's F-key buttons split each entry into a bold
	// key badge (always the brand color, so hotkeys are one scannable
	// column regardless of what they do) and a plain-text label that sits
	// flush on the bar itself — quieter than a wall of same-toned pills,
	// and disabled entries fade to Dim/Muted instead of competing for
	// attention.
	s.KeyBadge = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.HighlightFg).
		Background(p.Primary).
		Padding(0, 1)
	s.KeyBadgeOff = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Muted).
		Background(p.Dim).
		Padding(0, 1)
	s.KeyLabel = lipgloss.NewStyle().
		Foreground(p.Foreground).
		Background(p.BorderBlur).
		Padding(0, 1)
	s.KeyLabelOff = lipgloss.NewStyle().
		Foreground(p.Muted).
		Background(p.BorderBlur).
		Padding(0, 1)

	s.PanelFocused = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.BorderFocus)

	s.PanelBlurred = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.BorderBlur)

	s.PanelTitle = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Primary)

	s.TreeGroup = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Secondary)

	s.TreeItem = lipgloss.NewStyle().
		Foreground(p.Foreground)

	s.TreeSelected = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.HighlightFg).
		Background(p.Highlight)

	s.TreeMuted = lipgloss.NewStyle().
		Foreground(p.Muted)

	s.DetailLabel = lipgloss.NewStyle().
		Foreground(p.Muted).
		Width(20)

	s.DetailValue = lipgloss.NewStyle().
		Foreground(p.Foreground).
		Bold(true)

	s.MetricTile = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.BorderBlur).
		Padding(0, 2).
		Align(lipgloss.Center)

	s.MetricLabel = lipgloss.NewStyle().
		Foreground(p.Muted)

	s.MetricValue = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Primary)

	s.ConnectedBadge = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Success)

	s.DisconnectedBad = lipgloss.NewStyle().
		Bold(true).
		Foreground(p.Error)

	s.Error = lipgloss.NewStyle().Foreground(p.Error).Bold(true)
	s.Success = lipgloss.NewStyle().Foreground(p.Success).Bold(true)
	s.Warning = lipgloss.NewStyle().Foreground(p.Warning).Bold(true)

	// Unfocused field labels stay at full brightness — dimming them would
	// mean ~90% of a form's labels read as gray at any moment, since only
	// one field is focused at a time. Focus is already communicated by the
	// "▸" prefix and accent color/bold applied at the call site.
	s.InputLabel = lipgloss.NewStyle().Foreground(p.Foreground)
	s.InputBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.BorderBlur).
		Padding(0, 1)
	s.InputFocus = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.BorderFocus).
		Padding(0, 1)

	return s
}
