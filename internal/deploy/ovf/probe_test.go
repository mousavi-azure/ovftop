package ovf

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func requireOvftool(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ovftool")
	if err != nil {
		t.Skip("ovftool not found on PATH, skipping probe test")
	}
	return path
}

// copyTestOVF stages the checked-in synthetic test.ovf alongside a
// same-sized dummy disk file in a temp dir, since ovftool's warnings
// (and some size calculations) depend on the referenced disk actually
// existing next to the descriptor.
func copyTestOVF(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	src, err := os.ReadFile("testdata/test.ovf")
	if err != nil {
		t.Fatalf("reading testdata/test.ovf: %v", err)
	}
	dst := filepath.Join(dir, "test.ovf")
	if err := os.WriteFile(dst, src, 0o644); err != nil {
		t.Fatalf("writing test.ovf: %v", err)
	}

	disk := make([]byte, 1024*1024)
	if err := os.WriteFile(filepath.Join(dir, "test-disk1.vmdk"), disk, 0o644); err != nil {
		t.Fatalf("writing test-disk1.vmdk: %v", err)
	}
	return dst
}

func TestProbeSyntheticOVF(t *testing.T) {
	ovftoolPath := requireOvftool(t)
	ovfPath := copyTestOVF(t)

	d, err := Probe(context.Background(), ovftoolPath, ovfPath)
	if err != nil {
		t.Fatalf("Probe: %v", err)
	}

	if d.Name != "Test Appliance" {
		t.Errorf("Name = %q, want Test Appliance", d.Name)
	}
	if d.Vendor != "Acme Corp" {
		t.Errorf("Vendor = %q, want Acme Corp", d.Vendor)
	}
	if d.Version != "1.0" || d.FullVersion != "1.0.0-build123" {
		t.Errorf("Version/FullVersion = %q/%q", d.Version, d.FullVersion)
	}
	if d.Description == "" {
		t.Error("expected non-empty Description (from Annotation)")
	}

	if len(d.Networks) != 1 || d.Networks[0].Name != "VM Network" {
		t.Fatalf("unexpected Networks: %+v", d.Networks)
	}

	if len(d.VMs) != 1 {
		t.Fatalf("expected 1 VM, got %d", len(d.VMs))
	}
	vm := d.VMs[0]
	if vm.NumCPUs != 2 {
		t.Errorf("NumCPUs = %d, want 2", vm.NumCPUs)
	}
	if vm.MemoryBytes != 4096*1024*1024 {
		t.Errorf("MemoryBytes = %d, want %d", vm.MemoryBytes, int64(4096*1024*1024))
	}
	if len(vm.Disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(vm.Disks))
	}
	if vm.Disks[0].CapacityBytes != 20*1024*1024*1024 {
		t.Errorf("disk capacity = %d, want 20GB", vm.Disks[0].CapacityBytes)
	}

	if len(d.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(d.Properties))
	}
	if d.Properties[0].Key != "hostname" || d.Properties[0].Default != "test-host" {
		t.Errorf("unexpected property[0]: %+v", d.Properties[0])
	}

	if len(d.Warnings) == 0 {
		t.Error("expected at least one warning (missing manifest / size mismatch)")
	}
}

func TestProbeMissingFile(t *testing.T) {
	ovftoolPath := requireOvftool(t)

	_, err := Probe(context.Background(), ovftoolPath, filepath.Join(t.TempDir(), "does-not-exist.ovf"))
	if err == nil {
		t.Fatal("expected an error probing a nonexistent file")
	}
}
