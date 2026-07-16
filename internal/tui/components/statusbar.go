package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

// KeyHint is one F-key (or Ctrl-key) entry in the bottom status bar.
type KeyHint struct {
	Key     string
	Label   string
	Enabled bool
}

// RenderStatusBar draws the Midnight-Commander-style bottom bar as a row of
// buttons, each split into a bold key badge (F1, F2, ...) and a plain-text
// label. Keeping every badge in the same brand color makes the hotkey
// column scannable at a glance, while labels sit flush on the bar so they
// don't compete with it; disabled buttons fade to a muted tone instead of
// being hidden, and a hairline gap between buttons keeps them from
// visually merging into a single block.
func RenderStatusBar(styles theme.Styles, width int, hints []KeyHint) string {
	full := renderStatusBarButtons(styles, hints, true)
	// The full key+label row doesn't fit every terminal width. Rather than
	// hard-truncating mid-button (which used to cut labels off right after
	// their badge, e.g. "F6" with "Export" silently chopped away), fall
	// back to a badge-only row once the labeled version overflows — every
	// button stays intact, just without its text.
	if width <= 0 || lipgloss.Width(full) <= width {
		return styles.StatusBar.Width(width).Render(full)
	}
	compact := renderStatusBarButtons(styles, hints, false)
	return styles.StatusBar.Width(width).MaxWidth(width).Render(compact)
}

func renderStatusBarButtons(styles theme.Styles, hints []KeyHint, withLabels bool) string {
	var b strings.Builder
	for i, h := range hints {
		badge, label := styles.KeyBadge, styles.KeyLabel
		if !h.Enabled {
			badge, label = styles.KeyBadgeOff, styles.KeyLabelOff
		}
		b.WriteString(badge.Render(h.Key))
		if withLabels {
			b.WriteString(label.Render(h.Label))
		}
		if i < len(hints)-1 {
			b.WriteString(styles.StatusBar.Render(" "))
		}
	}
	return b.String()
}
