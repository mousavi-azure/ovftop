package ovf

// Descriptor is the UI-friendly view of an OVF/OVA package's metadata,
// derived from an ovftool probe.
type Descriptor struct {
	Name        string
	Vendor      string
	Version     string
	FullVersion string
	Description string

	DownloadSizeBytes int64
	FlatSizeBytes     int64
	SparseSizeUnknown bool
	SparseSizeBytes   int64

	Networks []Network
	VMs      []VM
	// Properties are OVF vApp properties (the standard mechanism appliances
	// use for deploy-time configuration: hostnames, static IPs, license
	// keys, etc.), aggregated across all virtual systems in the package.
	Properties []Property

	Warnings []string
}

// Network is one OVF-defined network that a deployment must map to a
// target network.
type Network struct {
	Name        string
	Description string
}

// VM describes one virtual system inside the package (almost always
// exactly one, except for multi-tier vApps).
type VM struct {
	Name           string
	OSID           string
	NumCPUs        int
	CoresPerSocket string
	MemoryBytes    int64
	HardwareFamily string
	Disks          []Disk
}

// Disk is one virtual disk declared by a virtual system.
type Disk struct {
	CapacityBytes int64
	Controller    string
}

// Property is one user-configurable OVF environment property.
type Property struct {
	Key         string
	Label       string
	Type        string
	Category    string
	Description string
	Default     string
}

// TotalCPUs sums vCPUs across all virtual systems in the package.
func (d *Descriptor) TotalCPUs() int {
	n := 0
	for _, vm := range d.VMs {
		n += vm.NumCPUs
	}
	return n
}

// TotalMemoryBytes sums memory across all virtual systems in the package.
func (d *Descriptor) TotalMemoryBytes() int64 {
	var n int64
	for _, vm := range d.VMs {
		n += vm.MemoryBytes
	}
	return n
}

// TotalDiskBytes sums declared disk capacity across all virtual systems.
func (d *Descriptor) TotalDiskBytes() int64 {
	var n int64
	for _, vm := range d.VMs {
		for _, disk := range vm.Disks {
			n += disk.CapacityBytes
		}
	}
	return n
}

func toDescriptor(pr *probeResult) *Descriptor {
	d := &Descriptor{
		Name:        pr.ProductInfo.Name,
		Vendor:      pr.ProductInfo.Vendor,
		Version:     pr.ProductInfo.Version,
		FullVersion: pr.ProductInfo.FullVersion,
		Description: pr.Annotation,
	}

	d.DownloadSizeBytes = parseInt64(pr.Sizes.Download)
	d.FlatSizeBytes = parseInt64(pr.Sizes.Flat)
	if pr.Sizes.Sparse == "Unknown" || pr.Sizes.Sparse == "" {
		d.SparseSizeUnknown = true
	} else {
		d.SparseSizeBytes = parseInt64(pr.Sizes.Sparse)
	}

	for _, n := range pr.Networks {
		d.Networks = append(d.Networks, Network{Name: n.Name, Description: n.Description})
	}

	for _, v := range pr.VMs {
		vm := VM{Name: v.Name}
		if len(v.OSIDs) > 0 {
			vm.OSID = v.OSIDs[0]
		}
		if len(v.VirtualHardwares) > 0 {
			hw := v.VirtualHardwares[0]
			vm.NumCPUs = hw.NumberOfCpus
			vm.CoresPerSocket = hw.CoresPerSocket
			vm.MemoryBytes = hw.MemoryBytes
			if len(hw.Families) > 0 {
				vm.HardwareFamily = hw.Families[0]
			}
			for _, disk := range hw.Disks {
				controller := ""
				if len(disk.Controllers) > 0 {
					controller = disk.Controllers[0]
				}
				vm.Disks = append(vm.Disks, Disk{CapacityBytes: disk.CapacityByte, Controller: controller})
			}
		}
		d.VMs = append(d.VMs, vm)
	}

	seen := make(map[string]bool)
	for _, p := range pr.Properties {
		if seen[p.Key] {
			continue
		}
		seen[p.Key] = true
		d.Properties = append(d.Properties, Property{
			Key:         p.Key,
			Label:       p.Label,
			Type:        p.Type,
			Category:    p.Category,
			Description: p.Description,
			Default:     p.Value,
		})
	}

	return d
}

func parseInt64(s string) int64 {
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0
		}
		n = n*10 + int64(r-'0')
	}
	return n
}
