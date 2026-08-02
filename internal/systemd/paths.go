package systemd

import (
	"fmt"
	"os"
	"path/filepath"
)

// UnitDir returns the directory where quadlet/systemd unit files are stored.
// In user mode it returns ~/.config/containers/systemd/.
// In system mode it returns /etc/containers/systemd/.
func UnitDir(system bool) (string, error) {
	if system {
		return "/etc/containers/systemd", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining user home directory: %w", err)
	}
	return filepath.Join(home, ".config", "containers", "systemd"), nil
}
