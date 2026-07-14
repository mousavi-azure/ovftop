package deploy_test

import (
	"context"
	"crypto/tls"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vmware/govmomi/simulator"

	"github.com/mousavi-azure/ovftop/internal/deploy"
)

// TestDeployAgainstSimulatedESXi drives a real ovftool binary through a
// full deploy against a simulated ESXi host, exercising BuildArgs, the
// moref-based datastore/network flags, and the streaming Run() reader
// end-to-end. It's the same validation performed manually while designing
// the args builder, kept as an automated regression test.
func TestDeployAgainstSimulatedESXi(t *testing.T) {
	ovftoolPath, err := deploy.LocateOVFTool()
	if err != nil {
		t.Skip("ovftool not found on PATH, skipping live deploy test")
	}

	model := simulator.ESX()
	defer model.Remove()
	if err := model.Create(); err != nil {
		t.Fatalf("model.Create: %v", err)
	}
	model.Service.TLS = new(tls.Config)
	server := model.Service.NewServer()
	defer server.Close()

	ds := model.Map().Any("Datastore")
	net := model.Map().Any("Network")

	port, _ := strconv.Atoi(server.URL.Port())
	password, _ := server.URL.User.Password()

	dir := t.TempDir()
	ovfBytes, err := os.ReadFile("ovf/testdata/test.ovf")
	if err != nil {
		t.Fatalf("reading testdata: %v", err)
	}
	ovfPath := filepath.Join(dir, "test.ovf")
	if err := os.WriteFile(ovfPath, ovfBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "test-disk1.vmdk"), make([]byte, 1024*1024), 0o644); err != nil {
		t.Fatal(err)
	}

	spec := deploy.Spec{
		SourcePath:   ovfPath,
		VMName:       "integration-test-vm",
		Hostname:     server.URL.Hostname(),
		Port:         port,
		Username:     server.URL.User.Username(),
		Password:     password,
		DiskMode:     deploy.DiskModeThin,
		DatastoreRef: ds.Reference(),
		Networks: []deploy.NetworkMapping{
			{OVFName: "VM Network", TargetRef: net.Reference()},
		},
		Properties: map[string]string{"hostname": "integration-host"},
	}

	args := deploy.BuildArgs(spec)

	var lines []string
	var doneSeen, success bool
	err = deploy.Run(context.Background(), ovftoolPath, args, func(e deploy.Event) {
		if e.Kind == deploy.EventLine {
			lines = append(lines, e.Text)
		} else {
			doneSeen = true
			success = e.Success
		}
	})

	if !doneSeen {
		t.Fatal("expected a terminal EventDone")
	}
	if !success || err != nil {
		t.Fatalf("expected successful deploy, err=%v, output=%v", err, lines)
	}

	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Completed successfully") {
		t.Errorf("expected 'Completed successfully' in output, got:\n%s", joined)
	}
}
