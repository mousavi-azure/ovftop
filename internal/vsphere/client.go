// Package vsphere wraps govmomi to connect to an ESXi host or vCenter
// Server and discover its inventory (datacenters, clusters, hosts,
// datastores, networks, resource pools, VMs and templates).
package vsphere

import (
	"context"
	"fmt"
	"net/url"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

// ConnectParams describes how to reach and authenticate against a vSphere
// endpoint. It is deliberately decoupled from config.ConnectionProfile so
// this package has no dependency on how credentials are persisted.
type ConnectParams struct {
	Hostname  string
	Port      int
	Username  string
	Password  string
	IgnoreSSL bool
}

func (p ConnectParams) url() (*url.URL, error) {
	host := p.Hostname
	if p.Port != 0 {
		host = fmt.Sprintf("%s:%d", p.Hostname, p.Port)
	}
	u, err := soap.ParseURL(host)
	if err != nil {
		return nil, fmt.Errorf("invalid host %q: %w", p.Hostname, err)
	}
	u.Path = "/sdk"
	u.User = url.UserPassword(p.Username, p.Password)
	return u, nil
}

// Client is a thin, discovery-focused wrapper around a govmomi session.
type Client struct {
	vim   *govmomi.Client
	about types.AboutInfo
}

// Connect logs into the target ESXi host or vCenter Server. The returned
// error is safe to display directly to a user (it does not leak the
// password, and wraps auth/network failures with the offending host).
func Connect(ctx context.Context, p ConnectParams) (*Client, error) {
	u, err := p.url()
	if err != nil {
		return nil, err
	}

	vim, err := govmomi.NewClient(ctx, u, p.IgnoreSSL)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", p.Hostname, err)
	}

	return &Client{vim: vim, about: vim.ServiceContent.About}, nil
}

// Close logs out of the session.
func (c *Client) Close(ctx context.Context) error {
	return c.vim.Logout(ctx)
}

// IsVC reports whether the connected endpoint is a vCenter Server (true)
// or a standalone ESXi host (false).
func (c *Client) IsVC() bool { return c.vim.IsVC() }

// About returns the connected endpoint's product/version metadata.
func (c *Client) About() types.AboutInfo { return c.about }

// CloneVM clones the VM/template identified by ref into a new VM named
// newName, placed in the same folder and resource pool as the source, and
// waits for the clone task to finish. The clone is left powered off
// regardless of the source's current power state.
func (c *Client) CloneVM(ctx context.Context, ref types.ManagedObjectReference, newName string) error {
	vm := object.NewVirtualMachine(c.vim.Client, ref)

	var mvm mo.VirtualMachine
	if err := vm.Properties(ctx, ref, []string{"parent", "resourcePool"}, &mvm); err != nil {
		return fmt.Errorf("looking up source VM: %w", err)
	}
	if mvm.Parent == nil {
		return fmt.Errorf("source VM has no parent folder")
	}
	folder := object.NewFolder(c.vim.Client, *mvm.Parent)

	spec := types.VirtualMachineCloneSpec{
		PowerOn:  false,
		Template: false,
	}
	if mvm.ResourcePool != nil {
		spec.Location.Pool = mvm.ResourcePool
	}

	task, err := vm.Clone(ctx, folder, newName, spec)
	if err != nil {
		return fmt.Errorf("starting clone task: %w", err)
	}
	if _, err := task.WaitForResult(ctx, nil); err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}
	return nil
}
