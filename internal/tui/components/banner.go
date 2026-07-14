package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/tui/theme"
	"github.com/mousavi-azure/ovftop/internal/version"
)

// logo is OVFTOP's ASCII wordmark (figlet, "big" font) — chosen over
// slanted/italic fonts because "V" and "F" collide into an illegible blob
// at small sizes in every slant variant; the upright block strokes here
// stay readable at a glance.
const logo = `  ______      ________ _______ ____  _____
 / __ \ \    / /  ____|__   __/ __ \|  __ \
| |  | \ \  / /| |__     | | | |  | | |__) |
| |  | |\ \/ / |  __|    | | | |  | |  ___/
| |__| | \  /  | |       | | | |__| | |
 \____/   \/   |_|       |_|  \____/|_|`

func logoWidth() int {
	w := 0
	for _, line := range strings.Split(logo, "\n") {
		if lw := lipgloss.Width(line); lw > w {
			w = lw
		}
	}
	return w
}

// RenderBanner draws the full ASCII wordmark inside a bordered panel, plus
// a tagline and author credit below it, falling back to a compact
// single-line wordmark on terminals too narrow for the figlet art.
func RenderBanner(styles theme.Styles, width int) string {
	tagline := styles.TreeMuted.Render("Terminal VMware Deployment Manager") +
		styles.TreeMuted.Render("  ·  v"+version.Version)
	link := lipgloss.NewStyle().Underline(true).Foreground(styles.Palette.Primary).Render("mousavi.dev")
	credit := styles.TreeMuted.Render("✨ Created by ") +
		styles.PanelTitle.Render("Mostafa Mousavi") +
		styles.TreeMuted.Render("  ·  🔗 ") + link

	var wordmark string
	if width > 0 && width < logoWidth()+6 {
		wordmark = styles.PanelTitle.Render("🖥  OVFTOP")
	} else {
		wordmark = lipgloss.NewStyle().Foreground(styles.Palette.Primary).Bold(true).Render(logo)
		wordmark = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Palette.Primary).
			Padding(0, 2).
			Render(wordmark)
	}

	block := lipgloss.JoinVertical(lipgloss.Center, wordmark, "", tagline, "", credit)

	if width > 0 {
		return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(block)
	}
	return block
}
