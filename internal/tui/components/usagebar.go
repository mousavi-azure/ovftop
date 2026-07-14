package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

// UsageBar is one row's worth of data for RenderUsageBars: a labeled
// used/total gauge (CPU, Memory, or a single datastore's capacity).
type UsageBar struct {
	Icon  string
	Label string
	Used  float64
	Total float64
	// Format renders the used/total pair for the trailing text, e.g.
	// "12.3/26.0 GHz" or "40/100 GB". Required.
	Format func(used, total float64) string
}

// RenderUsageBars draws a compact block-bar gauge per entry (btop-style),
// colored green/yellow/red by how full it is, inside a single bordered
// panel. width is the panel's outer width.
func RenderUsageBars(styles theme.Styles, width int, bars []UsageBar) string {
	if len(bars) == 0 {
		return ""
	}

	// Longest label decides where every bar's gauge starts, so the bars
	// themselves line up in a column regardless of label length.
	labelWidth := 0
	for _, b := range bars {
		l := len([]rune(b.Icon)) + 2 + len([]rune(b.Label))
		if l > labelWidth {
			labelWidth = l
		}
	}

	const trailingWidth = 24
	barWidth := width - labelWidth - trailingWidth - 6
	if barWidth < 10 {
		barWidth = 10
	}

	rows := make([]string, 0, len(bars))
	for _, b := range bars {
		pct := 0.0
		if b.Total > 0 {
			pct = b.Used / b.Total
		}
		if pct > 1 {
			pct = 1
		}
		if pct < 0 {
			pct = 0
		}

		fillStyle := styles.Success
		switch {
		case pct >= 0.9:
			fillStyle = styles.Error
		case pct >= 0.7:
			fillStyle = styles.Warning
		}

		filled := int(pct*float64(barWidth) + 0.5)
		if filled > barWidth {
			filled = barWidth
		}
		bar := fillStyle.Render(strings.Repeat("█", filled)) +
			styles.TreeMuted.Render(strings.Repeat("░", barWidth-filled))

		label := b.Icon + "  " + b.Label
		label = label + strings.Repeat(" ", labelWidth-len([]rune(label)))

		trailing := fmt.Sprintf("%3.0f%%  %s", pct*100, b.Format(b.Used, b.Total))

		rows = append(rows, styles.DetailValue.Render(label)+" ["+bar+"] "+styles.TreeMuted.Render(trailing))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return styles.PanelBlurred.Width(width - 2).Render(body)
}
