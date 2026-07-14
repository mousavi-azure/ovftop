package config

import "testing"

func TestSaveAndReloadDeploymentProfile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	profile := DeploymentProfile{
		Name:                "Ubuntu 24",
		ComputeResourceName: "Cluster0",
		Datastore:           &MorefRef{Type: "Datastore", Value: "datastore-1"},
		DiskMode:            "thin",
		PowerOn:             true,
		NetworkMap: map[string]MorefRef{
			"VM Network": {Type: "Network", Value: "network-1"},
		},
		Properties: map[string]string{"hostname": "ubuntu-01"},
	}

	saved, err := m.SaveDeploymentProfile(profile)
	if err != nil {
		t.Fatalf("SaveDeploymentProfile: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected generated ID")
	}

	m2, err := Load()
	if err != nil {
		t.Fatalf("reload Load: %v", err)
	}
	profiles := m2.DeploymentProfiles()
	if len(profiles) != 1 {
		t.Fatalf("expected 1 deployment profile after reload, got %d", len(profiles))
	}
	got := profiles[0]
	if got.Name != "Ubuntu 24" || got.ComputeResourceName != "Cluster0" {
		t.Errorf("unexpected profile after reload: %+v", got)
	}
	if got.Datastore == nil || got.Datastore.Value != "datastore-1" {
		t.Errorf("unexpected datastore ref: %+v", got.Datastore)
	}
	if got.NetworkMap["VM Network"].Value != "network-1" {
		t.Errorf("unexpected network map: %+v", got.NetworkMap)
	}
	if got.Properties["hostname"] != "ubuntu-01" {
		t.Errorf("unexpected properties: %+v", got.Properties)
	}

	if err := m2.DeleteDeploymentProfile(saved.ID); err != nil {
		t.Fatalf("DeleteDeploymentProfile: %v", err)
	}
	if got := len(m2.DeploymentProfiles()); got != 0 {
		t.Fatalf("expected 0 profiles after delete, got %d", got)
	}
}
