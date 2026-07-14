package deploy

import "github.com/vmware/govmomi/vim25/types"

// DiskMode is a target disk provisioning format, passed to ovftool's
// -dm/--diskMode flag.
type DiskMode string

const (
	DiskModeThin  DiskMode = "thin"
	DiskModeThick DiskMode = "thick"
)

// NetworkMapping assigns one OVF-declared network to a target network in
// the connected vSphere inventory.
type NetworkMapping struct {
	OVFName   string
	TargetRef types.ManagedObjectReference
}

// Spec captures every setting the deploy wizard collects, in a form that
// BuildArgs can turn into an ovftool invocation independent of any TUI
// concerns.
type Spec struct {
	SourcePath string
	VMName     string

	Hostname  string
	Port      int
	Username  string
	Password  string
	IgnoreSSL bool

	// DatacenterName and ComputeResourceName select the deploy target
	// using ovftool's documented `vi://.../<datacenter>/host/<name>` path
	// syntax. Both empty means "connect directly to an ESXi host with no
	// path", which the ovftool docs require for bare-ESXi targets.
	DatacenterName      string
	ComputeResourceName string
	ResourcePoolName    string // optional, nested pool under Resources

	// DatastoreRef and FolderRef are passed as ovftool morefs (via
	// --I:morefArgs), since we already have exact references from
	// discovery and morefs sidestep any name-ambiguity entirely.
	DatastoreRef types.ManagedObjectReference
	FolderRef    *types.ManagedObjectReference

	DiskMode         DiskMode
	PowerOn          bool
	ImportAsTemplate bool

	Networks   []NetworkMapping
	Properties map[string]string

	// ExtraArgs is a free-form, space-separated escape hatch for ovftool
	// flags the wizard doesn't have a dedicated field for.
	ExtraArgs string
}

// morefTypePrefix maps govmomi's short managed-object-reference Type (as
// used throughout internal/vsphere) to the fully-qualified vim namespace
// type ovftool's --I:morefArgs expects, e.g. "Datastore" -> "vim.Datastore".
var morefTypePrefix = map[string]string{
	"Datastore":                   "vim.Datastore",
	"Network":                     "vim.Network",
	"DistributedVirtualPortgroup": "vim.dvs.DistributedVirtualPortgroup",
	"ComputeResource":             "vim.ComputeResource",
	"ClusterComputeResource":      "vim.ClusterComputeResource",
	"HostSystem":                  "vim.HostSystem",
	"ResourcePool":                "vim.ResourcePool",
	"Folder":                      "vim.Folder",
}

// MorefArg formats a managed object reference as the "vim.Type:value"
// token ovftool expects when --I:morefArgs is set.
func MorefArg(ref types.ManagedObjectReference) string {
	prefix, ok := morefTypePrefix[ref.Type]
	if !ok {
		prefix = "vim." + ref.Type
	}
	return prefix + ":" + ref.Value
}
