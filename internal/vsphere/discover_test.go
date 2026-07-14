package vsphere

import (
	"context"
	"testing"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/simulator"
)

func TestDiscoverAgainstSimulatedESXi(t *testing.T) {
	model := simulator.ESX()
	defer model.Remove()
	if err := model.Create(); err != nil {
		t.Fatalf("model.Create: %v", err)
	}
	server := model.Service.NewServer()
	defer server.Close()

	ctx := context.Background()
	vim, err := govmomi.NewClient(ctx, server.URL, true)
	if err != nil {
		t.Fatalf("govmomi.NewClient: %v", err)
	}

	c := &Client{vim: vim, about: vim.ServiceContent.About}
	inv, err := c.Discover(ctx)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if inv.Summary.ConnectionType != "ESXi" {
		t.Errorf("expected ConnectionType ESXi, got %q", inv.Summary.ConnectionType)
	}
	if inv.Summary.DatacenterCount != 1 {
		t.Errorf("expected 1 datacenter, got %d", inv.Summary.DatacenterCount)
	}
	if inv.Summary.HostCount < 1 {
		t.Errorf("expected at least 1 host, got %d", inv.Summary.HostCount)
	}
	if inv.Summary.DatastoreCount < 1 {
		t.Errorf("expected at least 1 datastore, got %d", inv.Summary.DatastoreCount)
	}
	if inv.Summary.CPUCores == 0 {
		t.Error("expected non-zero CPU core count")
	}
	if inv.Summary.MemoryBytes == 0 {
		t.Error("expected non-zero memory size")
	}
	if len(inv.Root) != 1 {
		t.Fatalf("expected 1 root datacenter node, got %d", len(inv.Root))
	}

	dc := inv.Root[0]
	if dc.Kind != KindDatacenter {
		t.Errorf("expected root node kind datacenter, got %s", dc.Kind)
	}
	if len(dc.Children) != 5 {
		t.Fatalf("expected 5 group children (Hosts&Clusters/Datastores/Networks/ResourcePools/VMs), got %d", len(dc.Children))
	}
}

func TestDiscoverAgainstSimulatedVCenter(t *testing.T) {
	model := simulator.VPX()
	model.Datacenter = 1
	model.Cluster = 1
	model.Host = 0
	model.ClusterHost = 2
	model.Machine = 3
	defer model.Remove()
	if err := model.Create(); err != nil {
		t.Fatalf("model.Create: %v", err)
	}
	server := model.Service.NewServer()
	defer server.Close()

	ctx := context.Background()
	vim, err := govmomi.NewClient(ctx, server.URL, true)
	if err != nil {
		t.Fatalf("govmomi.NewClient: %v", err)
	}

	c := &Client{vim: vim, about: vim.ServiceContent.About}
	inv, err := c.Discover(ctx)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if inv.Summary.ConnectionType != "vCenter" {
		t.Errorf("expected ConnectionType vCenter, got %q", inv.Summary.ConnectionType)
	}
	if inv.Summary.VMCount != 3 {
		t.Errorf("expected 3 VMs, got %d", inv.Summary.VMCount)
	}
	if inv.Summary.HostCount != 2 {
		t.Errorf("expected 2 hosts, got %d", inv.Summary.HostCount)
	}

	dc := inv.Root[0]
	var clusterFound bool
	for _, group := range dc.Children {
		if group.Name != "Hosts & Clusters" {
			continue
		}
		for _, child := range group.Children {
			if child.Kind == KindCluster {
				clusterFound = true
				if len(child.Children) != 2 {
					t.Errorf("expected cluster to have 2 hosts, got %d", len(child.Children))
				}
			}
		}
	}
	if !clusterFound {
		t.Error("expected to find a cluster node under Hosts & Clusters")
	}
}
