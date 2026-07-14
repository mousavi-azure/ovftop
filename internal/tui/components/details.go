package components

import (
	"strings"

	"github.com/mousavi-azure/ovftop/internal/tui/theme"
	"github.com/mousavi-azure/ovftop/internal/vsphere"
)

// RenderDetails draws the right-hand details panel for whichever
// infrastructure node is currently selected in the tree.
func RenderDetails(styles theme.Styles, node *vsphere.Node) string {
	if node == nil {
		return styles.TreeMuted.Render("No selection")
	}

	title := nodeIcon(node) + " " + node.Name
	lines := []string{styles.PanelTitle.Render(title), ""}
	if len(node.Details) == 0 {
		lines = append(lines, styles.TreeMuted.Render("No additional details"))
	}
	for _, d := range node.Details {
		label := detailIcon(d.Label) + " " + d.Label
		lines = append(lines, styles.DetailLabel.Render(label)+styles.DetailValue.Render(d.Value))
	}
	return strings.Join(lines, "\n")
}

func detailIcon(label string) string {
	switch label {
	case "Vendor":
		return "🏭"
	case "Model":
		return "🧩"
	case "CPU Cores", "CPUs", "Total CPU":
		return "⚙ "
	case "CPU Speed":
		return "⚡"
	case "Memory", "Total Memory":
		return "🧠"
	case "Version":
		return "🔖"
	case "Build":
		return "🧱"
	case "Power State":
		return "🔌"
	case "Type":
		return "🗂"
	case "Capacity":
		return "💽"
	case "Free Space":
		return "📊"
	case "Accessible":
		return "✅"
	case "VMs", "Hosts":
		return "🖥"
	case "Guest OS":
		return "🐧"
	case "Name":
		return "🏷 "
	default:
		return "•"
	}
}
