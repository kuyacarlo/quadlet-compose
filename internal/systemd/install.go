package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Install writes unitContent to UnitDir/<name>.service and runs daemon-reload.
func Install(unitContent, name string, system bool) error {
	dir, err := UnitDir(system)
	if err != nil {
		return fmt.Errorf("install %s: %w", name, err)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("install %s: creating unit dir: %w", name, err)
	}

	path := filepath.Join(dir, name+".service")
	if err := os.WriteFile(path, []byte(unitContent), 0o644); err != nil {
		return fmt.Errorf("install %s: writing unit file: %w", name, err)
	}

	if err := daemonReload(system); err != nil {
		return fmt.Errorf("install %s: %w", name, err)
	}
	return nil
}

// daemonReload runs systemctl daemon-reload (with --user if not system mode).
func daemonReload(system bool) error {
	args := systemctlArgs(system, "daemon-reload")
	cmd := exec.Command("systemctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("daemon-reload: %w\n%s", err, output)
	}
	return nil
}

// systemctlArgs builds the argument slice for systemctl, prepending --user when needed.
func systemctlArgs(system bool, args ...string) []string {
	if system {
		return args
	}
	return append([]string{"--user"}, args...)
}
