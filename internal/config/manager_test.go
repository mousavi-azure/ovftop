package config

import "testing"

func TestSaveAndReloadProfile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	profile := ConnectionProfile{
		Name:      "Lab ESXi",
		Type:      ConnectionESXi,
		Hostname:  "esxi01.lab.local",
		Username:  "root",
		Port:      443,
		IgnoreSSL: true,
		Remember:  true,
	}

	saved, err := m.SaveProfile(profile, "s3cr3t")
	if err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected generated ID")
	}

	pw, ok, err := m.Password(saved.ID)
	if err != nil {
		t.Fatalf("Password: %v", err)
	}
	if !ok || pw != "s3cr3t" {
		t.Fatalf("expected vaulted password s3cr3t, got %q ok=%v", pw, ok)
	}

	// Reload from disk in a fresh Manager to prove persistence round-trips.
	m2, err := Load()
	if err != nil {
		t.Fatalf("reload Load: %v", err)
	}
	profiles := m2.Profiles()
	if len(profiles) != 1 || profiles[0].Hostname != "esxi01.lab.local" {
		t.Fatalf("unexpected profiles after reload: %+v", profiles)
	}

	pw2, ok, err := m2.Password(saved.ID)
	if err != nil {
		t.Fatalf("Password after reload: %v", err)
	}
	if !ok || pw2 != "s3cr3t" {
		t.Fatalf("expected vaulted password to survive reload, got %q ok=%v", pw2, ok)
	}
}

func TestForgetProfileDropsPassword(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	m, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	saved, err := m.SaveProfile(ConnectionProfile{Name: "x", Remember: true}, "hunter2")
	if err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	saved.Remember = false
	if _, err := m.SaveProfile(saved, ""); err != nil {
		t.Fatalf("SaveProfile update: %v", err)
	}

	_, ok, err := m.Password(saved.ID)
	if err != nil {
		t.Fatalf("Password: %v", err)
	}
	if ok {
		t.Fatal("expected password to be removed once Remember is false")
	}
}

func TestMasterPasswordDerivedVault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OVFTOP_MASTER_PASSWORD", "correct-horse-battery-staple")

	v, err := OpenVault(dir)
	if err != nil {
		t.Fatalf("OpenVault: %v", err)
	}
	if err := v.Set("p1", "secretpw"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	v2, err := OpenVault(dir)
	if err != nil {
		t.Fatalf("OpenVault reopen: %v", err)
	}
	pw, ok, err := v2.Get("p1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok || pw != "secretpw" {
		t.Fatalf("expected secretpw, got %q ok=%v", pw, ok)
	}

	t.Setenv("OVFTOP_MASTER_PASSWORD", "wrong-password")
	v3, err := OpenVault(dir)
	if err != nil {
		t.Fatalf("OpenVault wrong pass: %v", err)
	}
	if _, _, err := v3.Get("p1"); err == nil {
		t.Fatal("expected decryption failure with wrong master password")
	}
}
