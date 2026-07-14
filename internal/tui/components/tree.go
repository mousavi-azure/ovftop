package components

import (
	"strconv"
	"strings"

	"github.com/mousavi-azure/ovftop/internal/tui/theme"
	"github.com/mousavi-azure/ovftop/internal/vsphere"
)

// Tree renders vsphere.Node hierarchies as a Midnight-Commander-style
// expandable list: datacenters and category groups (Datastores, Networks,
// ...) are expanded by default, while clusters/hosts/VMs stay collapsed
// until the user opens them, keeping large environments readable.
type Tree struct {
	styles theme.Styles

	roots    []vsphere.Node
	expanded map[string]bool
	rows     []treeRow

	cursor int
	offset int
	width  int
	height int
}

type treeRow struct {
	node        *vsphere.Node
	depth       int
	path        string
	hasChildren bool
}

// NewTree creates an empty tree; call SetRoots once data is available.
func NewTree(styles theme.Styles) *Tree {
	return &Tree{styles: styles, expanded: map[string]bool{}}
}

// SetRoots replaces the tree's data and rebuilds the visible row list,
// preserving the user's expand/collapse choices where paths still match.
func (t *Tree) SetRoots(roots []vsphere.Node) {
	t.roots = roots
	t.rebuild()
}

// SetSize updates the viewport dimensions used for scrolling.
func (t *Tree) SetSize(width, height int) {
	t.width, t.height = width, height
	t.clampOffset()
}

// Selected returns the node under the cursor, or nil if the tree is empty.
func (t *Tree) Selected() *vsphere.Node {
	if t.cursor < 0 || t.cursor >= len(t.rows) {
		return nil
	}
	return t.rows[t.cursor].node
}

// SelectedDatacenter returns the name of the top-level datacenter node that
// contains the currently selected row (roots are always datacenter nodes),
// or "" if the tree is empty. Used to build ovftool vi:// locators, which
// need the datacenter a VM lives under.
func (t *Tree) SelectedDatacenter() string {
	if t.cursor < 0 || t.cursor >= len(t.rows) {
		return ""
	}
	path := t.rows[t.cursor].path
	idxStr := path
	if dot := strings.Index(path, "."); dot >= 0 {
		idxStr = path[:dot]
	}
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 || idx >= len(t.roots) {
		return ""
	}
	return t.roots[idx].Name
}

// MoveUp moves the cursor one visible row up.
func (t *Tree) MoveUp() {
	if t.cursor > 0 {
		t.cursor--
	}
	t.clampOffset()
}

// MoveDown moves the cursor one visible row down.
func (t *Tree) MoveDown() {
	if t.cursor < len(t.rows)-1 {
		t.cursor++
	}
	t.clampOffset()
}

// ToggleExpand expands or collapses the node under the cursor, if it has
// children.
func (t *Tree) ToggleExpand() {
	if t.cursor < 0 || t.cursor >= len(t.rows) {
		return
	}
	r := t.rows[t.cursor]
	if !r.hasChildren {
		return
	}
	t.expanded[r.path] = !t.isExpanded(r.node, r.path)
	t.rebuild()
}

func (t *Tree) isExpanded(n *vsphere.Node, path string) bool {
	if v, ok := t.expanded[path]; ok {
		return v
	}
	return n.Kind == vsphere.KindDatacenter || n.Kind == vsphere.KindGroup
}

func (t *Tree) rebuild() {
	selectedPath := ""
	if t.cursor >= 0 && t.cursor < len(t.rows) {
		selectedPath = t.rows[t.cursor].path
	}

	t.rows = nil
	var walk func(nodes []vsphere.Node, depth int, prefix string)
	walk = func(nodes []vsphere.Node, depth int, prefix string) {
		for i := range nodes {
			n := &nodes[i]
			path := prefix + strconv.Itoa(i)
			hasChildren := len(n.Children) > 0
			t.rows = append(t.rows, treeRow{node: n, depth: depth, path: path, hasChildren: hasChildren})
			if hasChildren && t.isExpanded(n, path) {
				walk(n.Children, depth+1, path+".")
			}
		}
	}
	walk(t.roots, 0, "")

	t.cursor = 0
	for i, r := range t.rows {
		if r.path == selectedPath {
			t.cursor = i
			break
		}
	}
	t.clampOffset()
}

func (t *Tree) clampOffset() {
	if t.height <= 0 {
		return
	}
	if t.cursor < t.offset {
		t.offset = t.cursor
	}
	if t.cursor >= t.offset+t.height {
		t.offset = t.cursor - t.height + 1
	}
	if t.offset < 0 {
		t.offset = 0
	}
}

// nodeIcon picks a glyph for a tree row. Hosts use colored squares and VMs
// use colored circles for their power state, so the shape alone tells you
// physical-vs-virtual at a glance and the color tells you power state.
func nodeIcon(n *vsphere.Node) string {
	switch n.Kind {
	case vsphere.KindDatacenter:
		return "🏢"
	case vsphere.KindCluster:
		return "🗂"
	case vsphere.KindHost:
		switch n.PowerState {
		case "poweredOn":
			return "🟩"
		case "poweredOff":
			return "🟥"
		default:
			return "🟨"
		}
	case vsphere.KindDatastore:
		return "💾"
	case vsphere.KindNetwork:
		return "🌐"
	case vsphere.KindResourcePool:
		return "📦"
	case vsphere.KindVM:
		switch n.PowerState {
		case "poweredOn":
			return "🟢"
		case "poweredOff":
			return "⚪"
		case "suspended":
			return "🟡"
		default:
			return "⚫"
		}
	case vsphere.KindTemplate:
		return "📄"
	case vsphere.KindGroup:
		return groupIcon(n.Name)
	default:
		return "📁"
	}
}

func groupIcon(name string) string {
	switch name {
	case "Hosts & Clusters":
		return "🖧"
	case "Datastores":
		return "💾"
	case "Networks":
		return "🌐"
	case "Resource Pools":
		return "📦"
	case "Virtual Machines":
		return "🖥"
	default:
		return "📁"
	}
}

// View renders the currently visible window of rows.
func (t *Tree) View() string {
	if len(t.rows) == 0 {
		return t.styles.TreeMuted.Render("(empty)")
	}

	end := t.offset + t.height
	if end > len(t.rows) || t.height <= 0 {
		end = len(t.rows)
	}

	var b strings.Builder
	for i := t.offset; i < end; i++ {
		r := t.rows[i]
		indent := strings.Repeat("  ", r.depth)

		arrow := "  "
		if r.hasChildren {
			if t.isExpanded(r.node, r.path) {
				arrow = "▾ "
			} else {
				arrow = "▸ "
			}
		}

		line := indent + arrow + nodeIcon(r.node) + " " + r.node.Name

		style := t.styles.TreeItem
		if r.node.Kind == vsphere.KindGroup {
			style = t.styles.TreeGroup
		}
		if i == t.cursor {
			style = t.styles.TreeSelected
		}
		if t.width > 0 {
			line = padOrTrim(line, t.width)
		}

		b.WriteString(style.Render(line))
		if i < end-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func padOrTrim(s string, width int) string {
	r := []rune(s)
	if len(r) > width {
		if width <= 1 {
			return string(r[:width])
		}
		return string(r[:width-1]) + "…"
	}
	return s + strings.Repeat(" ", width-len(r))
}
