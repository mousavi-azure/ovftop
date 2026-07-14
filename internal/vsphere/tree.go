package vsphere

import (
	"context"
	"strconv"

	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

// findAllIn is like findAll but scoped to a specific container (e.g. a
// datacenter's host/datastore/network/vm folder) rather than the whole
// service root.
func findAllIn(ctx context.Context, client *vim25.Client, root types.ManagedObjectReference, kind string, props []string, dst any) error {
	m := view.NewManager(client)
	cv, err := m.CreateContainerView(ctx, root, []string{kind}, true)
	if err != nil {
		return err
	}
	defer func() { _ = cv.Destroy(ctx) }()

	return cv.Retrieve(ctx, []string{kind}, props, dst)
}

func discoverDatacenter(ctx context.Context, client *vim25.Client, dc mo.Datacenter) (Node, []DatastoreUsage, error) {
	node := Node{Kind: KindDatacenter, Name: dc.Name, Ref: dc.Self}

	hostsGroup, err := discoverHostsAndClusters(ctx, client, dc.HostFolder)
	if err != nil {
		return node, nil, err
	}
	dsGroup, dsUsage, err := discoverDatastores(ctx, client, dc.DatastoreFolder)
	if err != nil {
		return node, nil, err
	}
	netGroup, err := discoverNetworks(ctx, client, dc.NetworkFolder)
	if err != nil {
		return node, nil, err
	}
	rpGroup, err := discoverResourcePools(ctx, client, dc.HostFolder)
	if err != nil {
		return node, nil, err
	}
	vmGroup, err := discoverVirtualMachines(ctx, client, dc.VmFolder)
	if err != nil {
		return node, nil, err
	}

	node.Children = []Node{hostsGroup, dsGroup, netGroup, rpGroup, vmGroup}
	return node, dsUsage, nil
}

func discoverHostsAndClusters(ctx context.Context, client *vim25.Client, root types.ManagedObjectReference) (Node, error) {
	group := Node{Kind: KindGroup, Name: "Hosts & Clusters"}

	var crs []mo.ComputeResource
	if err := findAllIn(ctx, client, root, "ComputeResource", []string{"name", "host", "summary"}, &crs); err != nil {
		return group, err
	}

	var hosts []mo.HostSystem
	if err := findAllIn(ctx, client, root, "HostSystem", []string{"name", "hardware", "summary", "runtime"}, &hosts); err != nil {
		return group, err
	}
	hostsByRef := make(map[types.ManagedObjectReference]mo.HostSystem, len(hosts))
	for _, h := range hosts {
		hostsByRef[h.Self] = h
	}

	for _, cr := range crs {
		var hostNodes []Node
		for _, href := range cr.Host {
			if h, ok := hostsByRef[href]; ok {
				hostNodes = append(hostNodes, hostNode(h))
			}
		}

		if cr.Self.Type == "ClusterComputeResource" || len(cr.Host) > 1 {
			group.Children = append(group.Children, Node{
				Kind:     KindCluster,
				Name:     cr.Name,
				Ref:      cr.Self,
				Details:  computeResourceDetails(cr),
				Children: hostNodes,
			})
		} else {
			group.Children = append(group.Children, hostNodes...)
		}
	}
	return group, nil
}

func hostNode(h mo.HostSystem) Node {
	var details []Detail
	if h.Hardware != nil {
		details = append(details,
			Detail{"Vendor", h.Hardware.SystemInfo.Vendor},
			Detail{"Model", h.Hardware.SystemInfo.Model},
			Detail{"CPU Cores", strconv.Itoa(int(h.Hardware.CpuInfo.NumCpuCores))},
			Detail{"CPU Speed", formatMHzGHz(int32(h.Hardware.CpuInfo.Hz / 1000000))},
			Detail{"Memory", formatBytesGB(h.Hardware.MemorySize)},
		)
	}
	if h.Summary.Config.Product != nil {
		details = append(details,
			Detail{"Version", h.Summary.Config.Product.Version},
			Detail{"Build", h.Summary.Config.Product.Build},
		)
	}
	details = append(details, Detail{"Power State", string(h.Runtime.PowerState)})

	return Node{Kind: KindHost, Name: h.Name, Ref: h.Self, PowerState: string(h.Runtime.PowerState), Details: details}
}

func computeResourceDetails(cr mo.ComputeResource) []Detail {
	details := []Detail{{"Hosts", strconv.Itoa(len(cr.Host))}}
	if s := cr.Summary; s != nil {
		if cs := s.GetComputeResourceSummary(); cs != nil {
			details = append(details,
				Detail{"Total CPU", formatMHzGHz(cs.TotalCpu)},
				Detail{"Total Memory", formatBytesGB(cs.TotalMemory)},
				Detail{"CPU Cores", strconv.Itoa(int(cs.NumCpuCores))},
			)
		}
	}
	return details
}

func discoverDatastores(ctx context.Context, client *vim25.Client, root types.ManagedObjectReference) (Node, []DatastoreUsage, error) {
	group := Node{Kind: KindGroup, Name: "Datastores"}
	var list []mo.Datastore
	if err := findAllIn(ctx, client, root, "Datastore", []string{"name", "summary"}, &list); err != nil {
		return group, nil, err
	}
	var usage []DatastoreUsage
	for _, ds := range list {
		s := ds.Summary
		group.Children = append(group.Children, Node{
			Kind: KindDatastore,
			Name: ds.Name,
			Ref:  ds.Self,
			Details: []Detail{
				{"Type", s.Type},
				{"Capacity", formatBytesGB(s.Capacity)},
				{"Free Space", formatBytesGB(s.FreeSpace)},
				{"Accessible", yesNo(s.Accessible)},
			},
		})
		usage = append(usage, DatastoreUsage{Name: ds.Name, CapacityBytes: s.Capacity, FreeBytes: s.FreeSpace})
	}
	return group, usage, nil
}

func discoverNetworks(ctx context.Context, client *vim25.Client, root types.ManagedObjectReference) (Node, error) {
	group := Node{Kind: KindGroup, Name: "Networks"}

	var stdNets []mo.Network
	if err := findAllIn(ctx, client, root, "Network", []string{"name"}, &stdNets); err != nil {
		return group, err
	}
	for _, n := range stdNets {
		group.Children = append(group.Children, Node{Kind: KindNetwork, Name: n.Name, Ref: n.Self})
	}

	var dvPortgroups []mo.Network
	if err := findAllIn(ctx, client, root, "DistributedVirtualPortgroup", []string{"name"}, &dvPortgroups); err == nil {
		for _, n := range dvPortgroups {
			group.Children = append(group.Children, Node{Kind: KindNetwork, Name: n.Name + " (DVS)", Ref: n.Self})
		}
	}
	return group, nil
}

func discoverResourcePools(ctx context.Context, client *vim25.Client, root types.ManagedObjectReference) (Node, error) {
	group := Node{Kind: KindGroup, Name: "Resource Pools"}
	var list []mo.ResourcePool
	if err := findAllIn(ctx, client, root, "ResourcePool", []string{"name", "summary", "vm"}, &list); err != nil {
		return group, err
	}
	for _, rp := range list {
		details := []Detail{{"VMs", strconv.Itoa(len(rp.Vm))}}
		if rp.Summary != nil {
			if s := rp.Summary.GetResourcePoolSummary(); s != nil {
				details = append(details, Detail{"Name", s.Name})
			}
		}
		group.Children = append(group.Children, Node{
			Kind:    KindResourcePool,
			Name:    rp.Name,
			Ref:     rp.Self,
			Details: details,
		})
	}
	return group, nil
}

func discoverVirtualMachines(ctx context.Context, client *vim25.Client, root types.ManagedObjectReference) (Node, error) {
	group := Node{Kind: KindGroup, Name: "Virtual Machines"}
	var vms []mo.VirtualMachine
	if err := findAllIn(ctx, client, root, "VirtualMachine", []string{"name", "summary", "runtime"}, &vms); err != nil {
		return group, err
	}
	for _, vm := range vms {
		cfg := vm.Summary.Config
		kind := KindVM
		if cfg.Template {
			kind = KindTemplate
		}
		group.Children = append(group.Children, Node{
			Kind:       kind,
			Name:       vm.Name,
			Ref:        vm.Self,
			PowerState: string(vm.Runtime.PowerState),
			Details: []Detail{
				{"Guest OS", cfg.GuestFullName},
				{"CPUs", strconv.Itoa(int(cfg.NumCpu))},
				{"Memory", formatMBasGB(cfg.MemorySizeMB)},
				{"Power State", string(vm.Runtime.PowerState)},
			},
		})
	}
	return group, nil
}

func yesNo(b bool) string {
	if b {
		return "Yes"
	}
	return "No"
}
