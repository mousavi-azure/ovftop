package config

import (
	"strconv"
	"time"
)

// ConnectionType identifies whether a profile targets a standalone ESXi
// host or a vCenter Server instance.
type ConnectionType string

const (
	ConnectionESXi    ConnectionType = "esxi"
	ConnectionVCenter ConnectionType = "vcenter"
)

// ConnectionProfile describes a saved connection to an ESXi host or vCenter.
// The password itself is never stored here — it lives in the encrypted
// vault, keyed by ID, only when Remember is true.
type ConnectionProfile struct {
	ID            string         `yaml:"id"`
	Name          string         `yaml:"name"`
	Description   string         `yaml:"description,omitempty"`
	Type          ConnectionType `yaml:"type"`
	Hostname      string         `yaml:"hostname"`
	Username      string         `yaml:"username"`
	Port          int            `yaml:"port"`
	IgnoreSSL     bool           `yaml:"ignore_ssl"`
	Remember      bool           `yaml:"remember"`
	LastConnected time.Time      `yaml:"last_connected,omitempty"`
}

// Addr returns the host:port pair used to reach the target.
func (p ConnectionProfile) Addr() string {
	if p.Port == 0 {
		return p.Hostname
	}
	return p.Hostname + ":" + strconv.Itoa(p.Port)
}
