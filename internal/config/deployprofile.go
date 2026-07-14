package config

// MorefRef is a serializable managed object reference (type + value),
// used so deployment profiles can pin an exact datastore/network/folder
// without depending on names that might be renamed later.
type MorefRef struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

// DeploymentProfile is a reusable set of deploy-wizard target settings —
// e.g. "Ubuntu 24", "pfSense" — so repeat deployments of the same kind of
// appliance don't require re-entering datastore/network/property choices.
// It intentionally excludes anything connection- or source-specific
// (hostname, credentials, the OVA/OVF path itself).
type DeploymentProfile struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`

	ComputeResourceName string `yaml:"compute_resource_name,omitempty"`
	ResourcePoolName    string `yaml:"resource_pool_name,omitempty"`

	Datastore *MorefRef `yaml:"datastore,omitempty"`
	Folder    *MorefRef `yaml:"folder,omitempty"`

	DiskMode         string `yaml:"disk_mode,omitempty"`
	PowerOn          bool   `yaml:"power_on"`
	ImportAsTemplate bool   `yaml:"import_as_template"`

	// NetworkMap keys are OVF-declared network names.
	NetworkMap map[string]MorefRef `yaml:"network_map,omitempty"`
	Properties map[string]string   `yaml:"properties,omitempty"`
	ExtraArgs  string              `yaml:"extra_args,omitempty"`
}

// DeploymentProfiles returns saved deployment profiles.
func (m *Manager) DeploymentProfiles() []DeploymentProfile {
	out := make([]DeploymentProfile, len(m.data.DeploymentProfiles))
	copy(out, m.data.DeploymentProfiles)
	return out
}

// SaveDeploymentProfile inserts or updates a deployment profile. If
// profile.ID is empty a new ID is generated.
func (m *Manager) SaveDeploymentProfile(profile DeploymentProfile) (DeploymentProfile, error) {
	if profile.ID == "" {
		profile.ID = newProfileID()
	}

	found := false
	for i, p := range m.data.DeploymentProfiles {
		if p.ID == profile.ID {
			m.data.DeploymentProfiles[i] = profile
			found = true
			break
		}
	}
	if !found {
		m.data.DeploymentProfiles = append(m.data.DeploymentProfiles, profile)
	}

	if err := m.persist(); err != nil {
		return profile, err
	}
	return profile, nil
}

// DeleteDeploymentProfile removes a saved deployment profile.
func (m *Manager) DeleteDeploymentProfile(id string) error {
	filtered := m.data.DeploymentProfiles[:0]
	for _, p := range m.data.DeploymentProfiles {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	m.data.DeploymentProfiles = filtered
	return m.persist()
}
