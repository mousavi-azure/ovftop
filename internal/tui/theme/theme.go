// Package theme defines the application's color palettes and the derived
// lipgloss styles shared by every screen and component.
package theme

import "github.com/charmbracelet/lipgloss"

// Palette is a named set of colors. Screens never reference raw colors
// directly — they go through Styles, which is derived from a Palette.
type Palette struct {
	Name string

	Background lipgloss.Color
	Foreground lipgloss.Color
	// Muted is a secondary but still clearly readable text color (used for
	// descriptions, labels, timestamps). Dim is separate and specifically
	// for UI chrome fills (disabled button pills, subtle dividers) that
	// text itself never sits directly on top of — keeping them distinct
	// avoids a low-contrast pill where Muted text landed on a Muted fill.
	Muted lipgloss.Color
	Dim   lipgloss.Color

	Primary   lipgloss.Color // accent: titles, active borders, key hints
	Secondary lipgloss.Color // subdued accent: group headers
	Success   lipgloss.Color
	Warning   lipgloss.Color
	Error     lipgloss.Color

	Highlight   lipgloss.Color // selected row background
	HighlightFg lipgloss.Color // selected row foreground
	BorderFocus lipgloss.Color
	BorderBlur  lipgloss.Color
}

var Dark = Palette{
	Name:        "dark",
	Background:  lipgloss.Color("235"),
	Foreground:  lipgloss.Color("252"),
	Muted:       lipgloss.Color("250"),
	Dim:         lipgloss.Color("243"),
	Primary:     lipgloss.Color("39"),  // blue
	Secondary:   lipgloss.Color("214"), // amber
	Success:     lipgloss.Color("42"),
	Warning:     lipgloss.Color("214"),
	Error:       lipgloss.Color("203"),
	Highlight:   lipgloss.Color("39"),
	HighlightFg: lipgloss.Color("235"),
	BorderFocus: lipgloss.Color("39"),
	BorderBlur:  lipgloss.Color("240"),
}

var Dracula = Palette{
	Name:        "dracula",
	Background:  lipgloss.Color("#282a36"),
	Foreground:  lipgloss.Color("#f8f8f2"),
	Muted:       lipgloss.Color("#8a94b3"),
	Dim:         lipgloss.Color("#6272a4"),
	Primary:     lipgloss.Color("#bd93f9"),
	Secondary:   lipgloss.Color("#ffb86c"),
	Success:     lipgloss.Color("#50fa7b"),
	Warning:     lipgloss.Color("#f1fa8c"),
	Error:       lipgloss.Color("#ff5555"),
	Highlight:   lipgloss.Color("#bd93f9"),
	HighlightFg: lipgloss.Color("#282a36"),
	BorderFocus: lipgloss.Color("#bd93f9"),
	BorderBlur:  lipgloss.Color("#44475a"),
}

var Nord = Palette{
	Name:        "nord",
	Background:  lipgloss.Color("#2e3440"),
	Foreground:  lipgloss.Color("#eceff4"),
	Muted:       lipgloss.Color("#7b88a1"),
	Dim:         lipgloss.Color("#4c566a"),
	Primary:     lipgloss.Color("#88c0d0"),
	Secondary:   lipgloss.Color("#ebcb8b"),
	Success:     lipgloss.Color("#a3be8c"),
	Warning:     lipgloss.Color("#ebcb8b"),
	Error:       lipgloss.Color("#bf616a"),
	Highlight:   lipgloss.Color("#88c0d0"),
	HighlightFg: lipgloss.Color("#2e3440"),
	BorderFocus: lipgloss.Color("#88c0d0"),
	BorderBlur:  lipgloss.Color("#4c566a"),
}

var Light = Palette{
	Name:        "light",
	Background:  lipgloss.Color("255"),
	Foreground:  lipgloss.Color("235"),
	Muted:       lipgloss.Color("240"),
	Dim:         lipgloss.Color("247"),
	Primary:     lipgloss.Color("25"),
	Secondary:   lipgloss.Color("130"),
	Success:     lipgloss.Color("28"),
	Warning:     lipgloss.Color("130"),
	Error:       lipgloss.Color("160"),
	Highlight:   lipgloss.Color("25"),
	HighlightFg: lipgloss.Color("255"),
	BorderFocus: lipgloss.Color("25"),
	BorderBlur:  lipgloss.Color("250"),
}

// Registry maps saved preference names to palettes.
var Registry = map[string]Palette{
	"dark":    Dark,
	"light":   Light,
	"dracula": Dracula,
	"nord":    Nord,
}

// Names lists palette names in a stable, presentation-friendly order.
var Names = []string{"dark", "light", "dracula", "nord"}

// Get returns the named palette, falling back to Dark if unknown.
func Get(name string) Palette {
	if p, ok := Registry[name]; ok {
		return p
	}
	return Dark
}

// Next returns the palette name after current in Names, wrapping around
// (and defaulting to the first entry if current is unrecognized).
func Next(current string) string {
	for i, n := range Names {
		if n == current {
			return Names[(i+1)%len(Names)]
		}
	}
	return Names[0]
}
