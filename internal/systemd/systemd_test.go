package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnitDir_UserMode(t *testing.T) {
	dir, err := UnitDir(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".config", "containers", "systemd")
	if dir != expected {
		t.Errorf("UnitDir(false) = %q, want %q", dir, expected)
	}
}

func TestUnitDir_SystemMode(t *testing.T) {
	dir, err := UnitDir(true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "/etc/containers/systemd"
	if dir != expected {
		t.Errorf("UnitDir(true) = %q, want %q", dir, expected)
	}
}

func TestUnitDir_UserModeNoHome(t *testing.T) {
	// Unset HOME to trigger the error path in UnitDir.
	origHome := os.Getenv("HOME")
	os.Unsetenv("HOME")
	// Also unset USER-related env that UserHomeDir might check
	origUser := os.Getenv("USER")
	os.Unsetenv("USER")
	defer func() {
		os.Setenv("HOME", origHome)
		if origUser != "" {
			os.Setenv("USER", origUser)
		}
	}()

	// This may or may not error depending on whether the OS can resolve
	// the home directory via /etc/passwd. But at minimum, verify it doesn't panic.
	_, _ = UnitDir(false)
}

func TestParseManagedUnit_Managed(t *testing.T) {
	tmpDir := t.TempDir()
	unitPath := filepath.Join(tmpDir, "myapp.service")
	content := `# Managed by quadlet-compose
# Source: /srv/myapp/compose.yml
# Generated: 2026-08-01T12:00:00Z

[Unit]
Description=quadlet-compose: myapp
`
	if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	unit, ok, err := parseManagedUnit(unitPath, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected unit to be managed")
	}
	if unit.Name != "myapp" {
		t.Errorf("Name = %q, want %q", unit.Name, "myapp")
	}
	if unit.Source != "/srv/myapp/compose.yml" {
		t.Errorf("Source = %q, want %q", unit.Source, "/srv/myapp/compose.yml")
	}
}

func TestParseManagedUnit_NotManaged(t *testing.T) {
	tmpDir := t.TempDir()
	unitPath := filepath.Join(tmpDir, "other.service")
	content := `[Unit]
Description=some other service

[Service]
ExecStart=/usr/bin/something
`
	if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, ok, err := parseManagedUnit(unitPath, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected unit to NOT be managed")
	}
}

func TestParseManagedUnit_NoSource(t *testing.T) {
	tmpDir := t.TempDir()
	unitPath := filepath.Join(tmpDir, "nosource.service")
	content := `# Managed by quadlet-compose
# Generated: 2026-08-01T12:00:00Z

[Unit]
Description=quadlet-compose: nosource
`
	if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	unit, ok, err := parseManagedUnit(unitPath, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected unit to be managed")
	}
	if unit.Name != "nosource" {
		t.Errorf("Name = %q, want %q", unit.Name, "nosource")
	}
	if unit.Source != "" {
		t.Errorf("Source = %q, want empty", unit.Source)
	}
}

func TestParseManagedUnit_FileNotFound(t *testing.T) {
	_, _, err := parseManagedUnit("/nonexistent/path.service", false)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSystemctlArgs_UserMode(t *testing.T) {
	args := systemctlArgs(false, "enable", "--now", "myapp.service")
	expected := []string{"--user", "enable", "--now", "myapp.service"}
	if !sliceEqual(args, expected) {
		t.Errorf("systemctlArgs(false, ...) = %v, want %v", args, expected)
	}
}

func TestSystemctlArgs_SystemMode(t *testing.T) {
	args := systemctlArgs(true, "enable", "--now", "myapp.service")
	expected := []string{"enable", "--now", "myapp.service"}
	if !sliceEqual(args, expected) {
		t.Errorf("systemctlArgs(true, ...) = %v, want %v", args, expected)
	}
}

func TestListWithTempDir(t *testing.T) {
	// We can't easily override UnitDir, so we test the parseManagedUnit + list logic
	// by scanning a directory manually (matching what List does internally).
	tmpDir := t.TempDir()

	// Write managed unit
	managed := `# Managed by quadlet-compose
# Source: /srv/pihole/compose.yml
# Generated: 2026-08-01T12:00:00Z

[Unit]
Description=quadlet-compose: pihole
`
	if err := os.WriteFile(filepath.Join(tmpDir, "pihole.service"), []byte(managed), 0644); err != nil {
		t.Fatal(err)
	}

	// Write another managed unit
	managed2 := `# Managed by quadlet-compose
# Source: /srv/nginx/compose.yml

[Unit]
Description=quadlet-compose: nginx
`
	if err := os.WriteFile(filepath.Join(tmpDir, "nginx.service"), []byte(managed2), 0644); err != nil {
		t.Fatal(err)
	}

	// Write non-managed unit
	notManaged := `[Unit]
Description=something else

[Service]
ExecStart=/bin/true
`
	if err := os.WriteFile(filepath.Join(tmpDir, "other.service"), []byte(notManaged), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a non-service file (should be skipped)
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("not a unit"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory (should be skipped)
	if err := os.MkdirAll(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	// Manually scan like List does to verify parse logic
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	var units []ManagedUnit
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".service") {
			continue
		}
		path := filepath.Join(tmpDir, entry.Name())
		unit, ok, err := parseManagedUnit(path, false)
		if err != nil {
			t.Fatalf("error parsing %s: %v", entry.Name(), err)
		}
		if ok {
			units = append(units, unit)
		}
	}

	if len(units) != 2 {
		t.Fatalf("expected 2 managed units, got %d", len(units))
	}

	// Check both units are present (order depends on readdir)
	names := map[string]bool{}
	for _, u := range units {
		names[u.Name] = true
	}
	if !names["pihole"] {
		t.Error("expected pihole in managed units")
	}
	if !names["nginx"] {
		t.Error("expected nginx in managed units")
	}
}

func TestList_NonexistentDir(t *testing.T) {
	// List should return nil when the directory doesn't exist.
	// We test this by setting HOME to a temp dir that doesn't have .config/containers/systemd.
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	units, err := List(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if units != nil {
		t.Errorf("expected nil units for nonexistent dir, got %v", units)
	}
}

func TestList_WithManagedUnits(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create the unit directory
	unitDir := filepath.Join(tmpDir, ".config", "containers", "systemd")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Write a managed unit
	managed := `# Managed by quadlet-compose
# Source: /srv/pihole/compose.yml
# Generated: 2026-08-01T12:00:00Z

[Unit]
Description=quadlet-compose: pihole
`
	if err := os.WriteFile(filepath.Join(unitDir, "pihole.service"), []byte(managed), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a non-managed unit
	notManaged := `[Unit]
Description=something else

[Service]
ExecStart=/bin/true
`
	if err := os.WriteFile(filepath.Join(unitDir, "other.service"), []byte(notManaged), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a non-service file (should be skipped)
	if err := os.WriteFile(filepath.Join(unitDir, "notes.txt"), []byte("not a unit"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory (should be skipped)
	if err := os.MkdirAll(filepath.Join(unitDir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}

	units, err := List(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(units) != 1 {
		t.Fatalf("expected 1 managed unit, got %d: %+v", len(units), units)
	}

	if units[0].Name != "pihole" {
		t.Errorf("unit name = %q, want %q", units[0].Name, "pihole")
	}
	if units[0].Source != "/srv/pihole/compose.yml" {
		t.Errorf("unit source = %q, want %q", units[0].Source, "/srv/pihole/compose.yml")
	}
}

func TestList_EmptyDir(t *testing.T) {
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Create empty unit directory
	unitDir := filepath.Join(tmpDir, ".config", "containers", "systemd")
	if err := os.MkdirAll(unitDir, 0755); err != nil {
		t.Fatal(err)
	}

	units, err := List(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(units) != 0 {
		t.Errorf("expected 0 units, got %d", len(units))
	}
}

func TestInstallWritesFile(t *testing.T) {
	// Override HOME so Install writes to a temp dir.
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	unitContent := `# Managed by quadlet-compose
# Source: /srv/test/compose.yml

[Unit]
Description=quadlet-compose: test

[Service]
Type=oneshot
ExecStart=/usr/bin/podman compose up -d

[Install]
WantedBy=default.target
`

	// Install will fail at daemonReload (no systemctl), but the file should be written.
	err := Install(unitContent, "test", false)

	// We expect an error from daemon-reload since systemctl likely isn't available.
	// But verify the file was written regardless.
	expectedDir := filepath.Join(tmpDir, ".config", "containers", "systemd")
	expectedPath := filepath.Join(expectedDir, "test.service")

	data, readErr := os.ReadFile(expectedPath)
	if readErr != nil {
		// If daemonReload failed, the file should still exist.
		if err == nil {
			t.Fatal("expected error from daemon-reload in test env")
		}
		// Check if directory was at least created
		if _, statErr := os.Stat(expectedDir); statErr != nil {
			t.Fatalf("unit dir was not created: %v", statErr)
		}
		// Re-read the file
		data, readErr = os.ReadFile(expectedPath)
		if readErr != nil {
			t.Fatalf("unit file was not written: %v", readErr)
		}
	}

	if string(data) != unitContent {
		t.Errorf("written content mismatch.\nGot:\n%s\nWant:\n%s", string(data), unitContent)
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
