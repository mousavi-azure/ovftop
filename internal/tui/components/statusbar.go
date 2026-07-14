package components

import (
	"strings"

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
	var b strings.Builder
	for i, h := range hints {
		badge, label := styles.KeyBadge, styles.KeyLabel
		if !h.Enabled {
			badge, label = styles.KeyBadgeOff, styles.KeyLabelOff
		}
		b.WriteString(badge.Render(h.Key))
		b.WriteString(label.Render(h.Label))
		if i < len(hints)-1 {
			b.WriteString(styles.StatusBar.Render(" "))
		}
	}
	return styles.StatusBar.Width(width).MaxWidth(width).Render(b.String())
}
