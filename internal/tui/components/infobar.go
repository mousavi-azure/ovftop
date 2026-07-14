package components

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/mousavi-azure/ovftop/internal/tui/theme"
)

// Metric is one labeled tile in the top info bar (e.g. "CPU" / "64 Cores").
type Metric struct {
	Label string
	Value string
}

// RenderInfoBar draws the connection banner plus a row of metric tiles,
// matching the spec's "Connection / Host / Version / CPU / Memory / ..."
// summary block.
func RenderInfoBar(styles theme.Styles, connected bool, connectionLabel string, metrics []Metric) string {
	badge := styles.DisconnectedBad.Render("🔴 Disconnected")
	if connected {
		badge = styles.ConnectedBadge.Render("🟢 Connected to " + connectionLabel)
	}

	tiles := make([]string, 0, len(metrics))
	for _, m := range metrics {
		content := lipgloss.JoinVertical(lipgloss.Center,
			styles.MetricLabel.Render(m.Label),
			styles.MetricValue.Render(m.Value),
		)
		tiles = append(tiles, styles.MetricTile.Render(content))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, tiles...)
	return lipgloss.JoinVertical(lipgloss.Left, badge, row)
}
