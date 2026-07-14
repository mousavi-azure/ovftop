// Package config handles persisted application settings: connection
// profiles, preferences, and the encrypted credential vault.
package config

import (
	"os"
	"path/filepath"
)

const appDirName = "ovftop"

// Dir returns the OS-appropriate configuration directory for the
// application, creating it if it does not already exist.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, appDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}

func profilesPath(dir string) string   { return filepath.Join(dir, "config.yaml") }
func vaultPath(dir string) string      { return filepath.Join(dir, "vault.enc") }
func vaultSaltPath(dir string) string  { return filepath.Join(dir, "vault.salt") }
func machineKeyPath(dir string) string { return filepath.Join(dir, "vault.key") }
func logPath(dir string) string        { return filepath.Join(dir, "ovftop.log") }

// LogPath returns the path to the application log file.
func LogPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return logPath(dir), nil
}
