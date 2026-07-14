package deploy

import (
	"strings"
	"testing"

	"github.com/vmware/govmomi/vim25/types"
)

func TestTargetLocatorBareESXi(t *testing.T) {
	spec := Spec{Hostname: "esxi01.lab.local", Username: "root", Password: "s3cret"}
	got := TargetLocator(spec)
	want := "vi://root:s3cret@esxi01.lab.local/"
	if got != want {
		t.Errorf("TargetLocator = %q, want %q", got, want)
	}
}

func TestTargetLocatorVCenterPath(t *testing.T) {
	spec := Spec{
		Hostname:            "vcenter.lab.local",
		Port:                443,
		Username:            "administrator@vsphere.local",
		Password:            "pw",
		DatacenterName:      "DC0",
		ComputeResourceName: "Cluster0",
		ResourcePoolName:    "Prod",
	}
	got := TargetLocator(spec)
	want := "vi://administrator@vsphere.local:pw@vcenter.lab.local:443/DC0/host/Cluster0/Resources/Prod"
	if got != want {
		t.Errorf("TargetLocator = %q, want %q", got, want)
	}
}

func TestBuildArgsCoreFlags(t *testing.T) {
	spec := Spec{
		SourcePath: "/tmp/app.ova",
		VMName:     "myvm",
		Hostname:   "esxi01",
		Username:   "root",
		Password:   "pw",
		DiskMode:   DiskModeThin,
		DatastoreRef: types.ManagedObjectReference{
			Type: "Datastore", Value: "datastore-1",
		},
		PowerOn: true,
		Networks: []NetworkMapping{
			{OVFName: "VM Network", TargetRef: types.ManagedObjectReference{Type: "Network", Value: "network-1"}},
		},
		Properties: map[string]string{"hostname": "myhost"},
	}

	args := BuildArgs(spec)
	joined := strings.Join(args, " ")

	for _, want := range []string{
		"--noSSLVerify", "--acceptAllEulas", "--overwrite", "--I:morefArgs",
		"-n=myvm", "-dm=thin", "-ds=vim.Datastore:datastore-1", "--powerOn",
		"--net:VM Network=vim.Network:network-1",
		"--prop:hostname=myhost",
		"/tmp/app.ova",
		"vi://root:pw@esxi01/",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected args to contain %q, got: %s", want, joined)
		}
	}

	if args[len(args)-2] != spec.SourcePath {
		t.Errorf("expected source path as second-to-last arg, got %q", args[len(args)-2])
	}
	if args[len(args)-1] != TargetLocator(spec) {
		t.Errorf("expected target locator as last arg, got %q", args[len(args)-1])
	}
}

func TestBuildArgsImportAsTemplateAndFolder(t *testing.T) {
	folderRef := types.ManagedObjectReference{Type: "Folder", Value: "group-v3"}
	spec := Spec{
		SourcePath:       "app.ovf",
		Hostname:         "esxi01",
		ImportAsTemplate: true,
		FolderRef:        &folderRef,
	}
	joined := strings.Join(BuildArgs(spec), " ")
	if !strings.Contains(joined, "--importAsTemplate") {
		t.Error("expected --importAsTemplate")
	}
	if !strings.Contains(joined, "--vmFolder=vim.Folder:group-v3") {
		t.Error("expected --vmFolder=vim.Folder:group-v3")
	}
}

func TestBuildArgsExtraArgsAppended(t *testing.T) {
	spec := Spec{SourcePath: "app.ovf", Hostname: "esxi01", ExtraArgs: "--lax --diskSize:vm1=1024"}
	joined := strings.Join(BuildArgs(spec), " ")
	if !strings.Contains(joined, "--lax") || !strings.Contains(joined, "--diskSize:vm1=1024") {
		t.Errorf("expected extra args appended, got: %s", joined)
	}
}

func TestBuildArgsPropertiesAreSortedForDeterminism(t *testing.T) {
	spec := Spec{
		SourcePath: "app.ovf",
		Hostname:   "esxi01",
		Properties: map[string]string{"zeta": "1", "alpha": "2", "middle": "3"},
	}
	args := BuildArgs(spec)

	var order []string
	for _, a := range args {
		if strings.HasPrefix(a, "--prop:") {
			order = append(order, a)
		}
	}
	want := []string{"--prop:alpha=2", "--prop:middle=3", "--prop:zeta=1"}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Errorf("prop order[%d] = %q, want %q", i, order[i], want[i])
		}
	}
}

func TestRedactedCommandLineMasksPassword(t *testing.T) {
	line := RedactedCommandLine("/usr/local/bin/ovftool", []string{
		"--acceptAllEulas", "app.ovf", "vi://root:supersecret@esxi01/",
	})
	if strings.Contains(line, "supersecret") {
		t.Errorf("expected password to be redacted, got: %s", line)
	}
	if !strings.Contains(line, "vi://root:***@esxi01/") {
		t.Errorf("expected redacted locator, got: %s", line)
	}
}
