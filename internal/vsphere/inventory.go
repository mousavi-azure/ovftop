package vsphere

import (
	"context"
	"fmt"

	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// Kind identifies the category of an inventory node, used by the TUI to
// pick icons and detail layouts.
type Kind string

const (
	KindDatacenter   Kind = "datacenter"
	KindCluster      Kind = "cluster"
	KindHost         Kind = "host"
	KindDatastore    Kind = "datastore"
	KindNetwork      Kind = "network"
	KindResourcePool Kind = "resourcepool"
	KindVM           Kind = "vm"
	KindTemplate     Kind = "template"
	KindGroup        Kind = "group" // synthetic grouping node, e.g. "Datastores"
)

// Node is one entry in the infrastructure tree. Group nodes (Kind ==
// KindGroup) exist purely to organize the tree the way the spec's left
// panel expects (Datastores, Networks, ... as sibling categories).
type Node struct {
	Kind Kind
	Name string
	Ref  types.ManagedObjectReference
	// PowerState is the raw vSphere power state ("poweredOn", "poweredOff",
	// "suspended", "standBy") for Host/VM/Template nodes, used by the TUI
	// to render a colored status dot. Empty for nodes where it's not
	// meaningful (datacenters, groups, datastores, ...).
	PowerState string
	Details    []Detail
	Children   []Node
}

// Detail is a single label/value pair shown in the right-hand details
// panel for whichever node is selected.
type Detail struct {
	Label string
	Value string
}

// Summary holds the at-a-glance numbers shown in the main screen's top
// info bar.
type Summary struct {
	ConnectionType  string // "ESXi" or "vCenter"
	Host            string
	Version         string
	Build           string
	CPUCores        int32
	MemoryBytes     int64
	VMCount         int
	DatastoreCount  int
	NetworkCount    int
	TemplateCount   int
	DatacenterCount int
	HostCount       int

	// CPUUsageMHz/CPUTotalMHz and MemoryUsageBytes/MemoryBytes feed the
	// dashboard's usage bars. Usage comes from each host's QuickStats,
	// which vCenter/ESXi refresh every ~20s, so it's a good-enough live
	// gauge without needing the full performance-manager API.
	CPUUsageMHz      int32
	CPUTotalMHz      int32
	MemoryUsageBytes int64

	Datastores []DatastoreUsage
}

// DatastoreUsage is the raw capacity/free figures for one datastore, used
// to render its usage bar (Node.Details carries the same numbers already
// formatted as strings for the tree's detail panel).
type DatastoreUsage struct {
	Name          string
	CapacityBytes int64
	FreeBytes     int64
}

// Inventory is the fully discovered snapshot of a connected endpoint.
type Inventory struct {
	Summary Summary
	Root    []Node
}

// Discover walks the connected endpoint's inventory and returns a
// structured snapshot suitable for rendering in the infrastructure tree.
// It is safe to call from a background goroutine; all network I/O is
// bounded by ctx.
func (c *Client) Discover(ctx context.Context) (*Inventory, error) {
	client := c.vim

	var datacenters []mo.Datacenter
	if err := findAll(ctx, client.Client, "Datacenter", []string{"name", "hostFolder", "datastoreFolder", "networkFolder", "vmFolder"}, &datacenters); err != nil {
		return nil, fmt.Errorf("listing datacenters: %w", err)
	}

	inv := &Inventory{}
	inv.Summary.ConnectionType = "ESXi"
	if c.IsVC() {
		inv.Summary.ConnectionType = "vCenter"
	}
	inv.Summary.Host = c.about.Name
	inv.Summary.Version = c.about.Version
	inv.Summary.Build = c.about.Build
	inv.Summary.DatacenterCount = len(datacenters)

	for _, dc := range datacenters {
		node, dsUsage, err := discoverDatacenter(ctx, client.Client, dc)
		if err != nil {
			return nil, fmt.Errorf("discovering datacenter %s: %w", dc.Name, err)
		}
		inv.Root = append(inv.Root, node)
		inv.Summary.Datastores = append(inv.Summary.Datastores, dsUsage...)

		for _, group := range node.Children {
			if group.Kind != KindGroup {
				continue
			}
			switch group.Name {
			case "Datastores":
				inv.Summary.DatastoreCount += len(group.Children)
			case "Networks":
				inv.Summary.NetworkCount += len(group.Children)
			case "Hosts & Clusters":
				inv.Summary.HostCount += countHostsRecursive(group.Children)
			case "Virtual Machines":
				inv.Summary.VMCount += countKind(group.Children, KindVM)
				inv.Summary.TemplateCount += countKind(group.Children, KindTemplate)
			}
		}
	}

	// CPU/memory totals and live usage are read directly from hosts, since
	// they're not otherwise rolled up anywhere per-datacenter.
	var hosts []mo.HostSystem
	if err := findAll(ctx, client.Client, "HostSystem", []string{"name", "hardware", "summary"}, &hosts); err == nil {
		for _, h := range hosts {
			if h.Hardware != nil {
				inv.Summary.CPUCores += int32(h.Hardware.CpuInfo.NumCpuCores)
				inv.Summary.MemoryBytes += h.Hardware.MemorySize
				cpuMHz := int32(h.Hardware.CpuInfo.Hz / 1000000)
				inv.Summary.CPUTotalMHz += cpuMHz * int32(h.Hardware.CpuInfo.NumCpuCores)
			}
			qs := h.Summary.QuickStats
			inv.Summary.CPUUsageMHz += qs.OverallCpuUsage
			inv.Summary.MemoryUsageBytes += int64(qs.OverallMemoryUsage) * 1024 * 1024
		}
	}

	return inv, nil
}

func countKind(nodes []Node, k Kind) int {
	n := 0
	for _, c := range nodes {
		if c.Kind == k {
			n++
		}
	}
	return n
}

func countHostsRecursive(nodes []Node) int {
	n := 0
	for _, c := range nodes {
		if c.Kind == KindHost {
			n++
		}
		n += countHostsRecursive(c.Children)
	}
	return n
}

// findAll retrieves every object of the given managed object type visible
// from the service's root folder, using a recursive container view.
func findAll(ctx context.Context, client *vim25.Client, kind string, props []string, dst any) error {
	m := view.NewManager(client)
	cv, err := m.CreateContainerView(ctx, client.ServiceContent.RootFolder, []string{kind}, true)
	if err != nil {
		return err
	}
	defer func() { _ = cv.Destroy(ctx) }()

	return cv.Retrieve(ctx, []string{kind}, props, dst)
}
