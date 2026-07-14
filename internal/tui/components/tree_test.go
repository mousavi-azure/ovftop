package components

import (
	"testing"

	"github.com/mousavi-azure/ovftop/internal/tui/theme"
	"github.com/mousavi-azure/ovftop/internal/vsphere"
)

func sampleTree() []vsphere.Node {
	return []vsphere.Node{
		{
			Kind: vsphere.KindDatacenter,
			Name: "DC0",
			Children: []vsphere.Node{
				{
					Kind: vsphere.KindGroup,
					Name: "Hosts & Clusters",
					Children: []vsphere.Node{
						{Kind: vsphere.KindHost, Name: "esxi01"},
						{Kind: vsphere.KindHost, Name: "esxi02"},
					},
				},
				{
					Kind: vsphere.KindGroup,
					Name: "Virtual Machines",
					Children: []vsphere.Node{
						{Kind: vsphere.KindVM, Name: "web01"},
					},
				},
			},
		},
	}
}

func TestTreeDefaultExpansion(t *testing.T) {
	tr := NewTree(theme.New(theme.Dark))
	tr.SetRoots(sampleTree())

	// Datacenter and its two groups are expanded by default; hosts/VMs are
	// visible immediately below their group without any user action.
	// DC0, Hosts & Clusters, esxi01, esxi02, Virtual Machines, web01.
	if got := len(tr.rows); got != 6 {
		t.Fatalf("expected 6 visible rows by default, got %d", got)
	}
	if tr.rows[0].node.Name != "DC0" {
		t.Errorf("expected row 0 to be DC0, got %s", tr.rows[0].node.Name)
	}
	if tr.rows[1].node.Name != "Hosts & Clusters" {
		t.Errorf("expected row 1 to be Hosts & Clusters, got %s", tr.rows[1].node.Name)
	}
}

func TestTreeMoveAndSelect(t *testing.T) {
	tr := NewTree(theme.New(theme.Dark))
	tr.SetRoots(sampleTree())

	if sel := tr.Selected(); sel == nil || sel.Name != "DC0" {
		t.Fatalf("expected initial selection DC0, got %+v", sel)
	}
	tr.MoveDown()
	if sel := tr.Selected(); sel == nil || sel.Name != "Hosts & Clusters" {
		t.Fatalf("expected selection Hosts & Clusters after MoveDown, got %+v", sel)
	}
	tr.MoveDown()
	if sel := tr.Selected(); sel == nil || sel.Name != "esxi01" {
		t.Fatalf("expected selection esxi01, got %+v", sel)
	}
	tr.MoveUp()
	tr.MoveUp()
	if sel := tr.Selected(); sel == nil || sel.Name != "DC0" {
		t.Fatalf("expected selection back at DC0, got %+v", sel)
	}
}

func TestTreeToggleCollapsesChildren(t *testing.T) {
	tr := NewTree(theme.New(theme.Dark))
	tr.SetRoots(sampleTree())

	tr.MoveDown() // Hosts & Clusters
	tr.ToggleExpand()
	// DC0, Hosts & Clusters (collapsed), Virtual Machines, web01.
	if got := len(tr.rows); got != 4 {
		t.Fatalf("expected 4 visible rows after collapsing Hosts & Clusters, got %d", got)
	}

	tr.ToggleExpand()
	if got := len(tr.rows); got != 6 {
		t.Fatalf("expected 6 visible rows after re-expanding, got %d", got)
	}
}

func TestTreeMoveBoundsDoNotPanic(t *testing.T) {
	tr := NewTree(theme.New(theme.Dark))
	tr.SetRoots(nil)
	tr.MoveDown()
	tr.MoveUp()
	tr.ToggleExpand()
	if sel := tr.Selected(); sel != nil {
		t.Fatalf("expected nil selection on empty tree, got %+v", sel)
	}
}
