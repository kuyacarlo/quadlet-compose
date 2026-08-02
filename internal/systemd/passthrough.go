package systemd

import (
	"fmt"
	"os"
	"os/exec"
)

// Status runs systemctl status for the given unit, piping output to stdout/stderr.
func Status(name string, system bool) error {
	args := systemctlArgs(system, "status", name+".service")
	cmd := exec.Command("systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("status %s: %w", name, err)
	}
	return nil
}

// Logs runs journalctl -u <name>.service, passing any extra arguments.
// Output is piped directly to stdout/stderr.
func Logs(name string, system bool, extraArgs []string) error {
	args := []string{"-u", name + ".service"}
	if !system {
		args = append(args, "--user")
	}
	args = append(args, extraArgs...)
	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("logs %s: %w", name, err)
	}
	return nil
}
