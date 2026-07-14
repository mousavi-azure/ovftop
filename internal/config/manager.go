package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

// Preferences holds cross-cutting UI settings independent of any single
// connection.
type Preferences struct {
	Theme string `yaml:"theme"`
	// RefreshIntervalSeconds is how often the dashboard auto-refreshes its
	// inventory. 0 disables auto-refresh entirely.
	RefreshIntervalSeconds int `yaml:"refresh_interval_seconds"`
}

// DefaultRefreshIntervalSeconds is the auto-refresh cadence used for new
// installs (no config.yaml yet) or configs predating this setting.
const DefaultRefreshIntervalSeconds = 120

// fileFormat is the on-disk shape of config.yaml.
type fileFormat struct {
	Preferences        Preferences         `yaml:"preferences"`
	Profiles           []ConnectionProfile `yaml:"profiles"`
	DeploymentProfiles []DeploymentProfile `yaml:"deployment_profiles"`
}

// Manager is the single entry point for reading and writing persisted
// profiles, preferences, and vaulted credentials.
type Manager struct {
	dir   string
	vault *Vault
	data  fileFormat
}

// Load reads config.yaml (creating an empty one if absent) and opens the
// credential vault.
func Load() (*Manager, error) {
	dir, err := Dir()
	if err != nil {
		return nil, fmt.Errorf("resolving config dir: %w", err)
	}

	m := &Manager{dir: dir}
	m.data.Preferences.Theme = "dark"
	m.data.Preferences.RefreshIntervalSeconds = DefaultRefreshIntervalSeconds

	if b, err := os.ReadFile(profilesPath(dir)); err == nil {
		if err := yaml.Unmarshal(b, &m.data); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", profilesPath(dir), err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	v, err := OpenVault(dir)
	if err != nil {
		return nil, fmt.Errorf("opening credential vault: %w", err)
	}
	m.vault = v
	return m, nil
}

func (m *Manager) persist() error {
	b, err := yaml.Marshal(&m.data)
	if err != nil {
		return err
	}
	return os.WriteFile(profilesPath(m.dir), b, 0o600)
}

// Profiles returns saved connection profiles, most recently used first.
func (m *Manager) Profiles() []ConnectionProfile {
	out := make([]ConnectionProfile, len(m.data.Profiles))
	copy(out, m.data.Profiles)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].LastConnected.After(out[j].LastConnected)
	})
	return out
}

// Preferences returns the current UI preferences.
func (m *Manager) Preferences() Preferences { return m.data.Preferences }

// SetPreferences persists updated UI preferences.
func (m *Manager) SetPreferences(p Preferences) error {
	m.data.Preferences = p
	return m.persist()
}

func newProfileID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// SaveProfile inserts or updates a profile. If profile.ID is empty a new ID
// is generated. When remember is true and password is non-empty, the
// password is written to the encrypted vault; otherwise any existing
// vaulted password for this profile is left untouched by this call.
func (m *Manager) SaveProfile(profile ConnectionProfile, password string) (ConnectionProfile, error) {
	if profile.ID == "" {
		profile.ID = newProfileID()
	}

	found := false
	for i, p := range m.data.Profiles {
		if p.ID == profile.ID {
			m.data.Profiles[i] = profile
			found = true
			break
		}
	}
	if !found {
		m.data.Profiles = append(m.data.Profiles, profile)
	}

	if err := m.persist(); err != nil {
		return profile, err
	}

	if profile.Remember && password != "" {
		if err := m.vault.Set(profile.ID, password); err != nil {
			return profile, fmt.Errorf("saving credentials: %w", err)
		}
	} else if !profile.Remember {
		_ = m.vault.Delete(profile.ID)
	}
	return profile, nil
}

// DeleteProfile removes a profile and its vaulted credentials, if any.
func (m *Manager) DeleteProfile(id string) error {
	filtered := m.data.Profiles[:0]
	for _, p := range m.data.Profiles {
		if p.ID != id {
			filtered = append(filtered, p)
		}
	}
	m.data.Profiles = filtered
	if err := m.persist(); err != nil {
		return err
	}
	return m.vault.Delete(id)
}

// Password returns the vaulted password for a profile, if one was saved.
func (m *Manager) Password(profileID string) (string, bool, error) {
	return m.vault.Get(profileID)
}
