package systemd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ManagedUnit represents a unit file managed by quadlet-compose.
type ManagedUnit struct {
	Name   string // unit name without .service suffix
	Source string // source compose file parsed from comment
	Status string // active, inactive, failed, etc.
}

// Enable runs systemctl enable --now for the given unit.
func Enable(name string, system bool) error {
	args := systemctlArgs(system, "enable", "--now", name+".service")
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("enable %s: %w\n%s", name, err, output)
	}
	return nil
}

// Disable runs systemctl disable --now for the given unit.
func Disable(name string, system bool) error {
	args := systemctlArgs(system, "disable", "--now", name+".service")
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("disable %s: %w\n%s", name, err, output)
	}
	return nil
}

// Remove disables the unit, deletes the unit file, and runs daemon-reload.
func Remove(name string, system bool) error {
	// Best-effort disable; unit may already be inactive.
	_ = Disable(name, system)

	dir, err := UnitDir(system)
	if err != nil {
		return fmt.Errorf("remove %s: %w", name, err)
	}

	path := filepath.Join(dir, name+".service")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s: deleting unit file: %w", name, err)
	}

	if err := daemonReload(system); err != nil {
		return fmt.Errorf("remove %s: %w", name, err)
	}
	return nil
}

// List scans the unit directory for files with a "# Managed by quadlet-compose" header
// and returns information about each managed unit.
func List(system bool) ([]ManagedUnit, error) {
	dir, err := UnitDir(system)
	if err != nil {
		return nil, fmt.Errorf("list units: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list units: reading dir: %w", err)
	}

	var units []ManagedUnit
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".service") {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		unit, ok, err := parseManagedUnit(path, system)
		if err != nil {
			return nil, fmt.Errorf("list units: %w", err)
		}
		if ok {
			units = append(units, unit)
		}
	}
	return units, nil
}

// parseManagedUnit reads the first lines of a unit file looking for the
// managed-by-quadlet-compose marker and optional source comment. Returns (unit, true, nil)
// if managed, (zero, false, nil) if not managed, or an error.
func parseManagedUnit(path string, system bool) (ManagedUnit, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return ManagedUnit{}, false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	managed := false
	source := ""

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "#") {
			break
		}
		if strings.TrimSpace(line) == "# Managed by quadlet-compose" {
			managed = true
		}
		if strings.HasPrefix(line, "# Source: ") {
			source = strings.TrimPrefix(line, "# Source: ")
		}
	}

	if !managed {
		return ManagedUnit{}, false, nil
	}

	name := strings.TrimSuffix(filepath.Base(path), ".service")
	status := unitStatus(name, system)

	return ManagedUnit{
		Name:   name,
		Source: source,
		Status: status,
	}, true, nil
}

// unitStatus returns the active state of a unit via systemctl is-active.
func unitStatus(name string, system bool) string {
	args := systemctlArgs(system, "is-active", name+".service")
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// is-active exits non-zero for inactive/failed; output still has the state.
		return strings.TrimSpace(string(output))
	}
	return strings.TrimSpace(string(output))
}
