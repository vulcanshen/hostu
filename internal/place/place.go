// Package place says where hostu keeps its things (tdp E3) and which hosts
// file it writes, each moved by its environment variable (tdp E2).
package place

import (
	"os"
	"path/filepath"

	"github.com/vulcanshen/hostu/internal/hosts"
)

// Config is the settings directory: config.yaml.
func Config() (string, error) {
	if d := os.Getenv("HOSTU__CONFIG"); d != "" {
		return d, nil
	}
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "hostu"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "hostu"), nil
}

// Data is the data directory: profiles.yaml and hosts.original.
func Data() (string, error) {
	if d := os.Getenv("HOSTU__DATA"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".hostu"), nil
}

// HostsFile is the hosts file hostu reads and writes.
func HostsFile() string {
	if f := os.Getenv("HOSTU__HOSTS_FILE"); f != "" {
		return f
	}
	return hosts.DefaultPath()
}
