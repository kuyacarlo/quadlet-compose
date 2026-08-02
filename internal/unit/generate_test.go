package unit

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name       string
		opts       Opts
		goldenFile string // relative to testdata/expected/
		contains   []string
		notContain []string
	}{
		{
			name: "simple case - no networks, no env files",
			opts: Opts{
				Name:            "myapp",
				ComposePath:     "/srv/myapp/docker-compose.yml",
				ComposeFilename: "docker-compose.yml",
				WorkingDir:      "/srv/myapp",
				ComposeBin:      "/usr/bin/podman compose",
				GeneratedAt:     fixedTime,
			},
			goldenFile: "simple.service",
			contains: []string{
				"# Managed by quadlet-compose",
				"# Source: /srv/myapp/docker-compose.yml",
				"# Generated: 2026-08-01T12:00:00Z",
				"Description=quadlet-compose: myapp",
				"After=network-online.target",
				"Type=oneshot",
				"RemainAfterExit=yes",
				"WorkingDirectory=/srv/myapp",
				"ExecStart=/usr/bin/podman compose -f docker-compose.yml up -d",
				"ExecStop=/usr/bin/podman compose -f docker-compose.yml down",
				"ExecReload=/usr/bin/podman compose -f docker-compose.yml up -d --force-recreate",
				"Restart=on-failure",
				"TimeoutStartSec=120",
				"WantedBy=default.target",
			},
			notContain: []string{
				"ExecStartPre=",
				"EnvironmentFile=",
			},
		},
		{
			name: "with external networks - verify /bin/sh -c wrapping",
			opts: Opts{
				Name:             "pihole",
				ComposePath:      "/srv/pihole/docker-compose.yml",
				ComposeFilename:  "docker-compose.yml",
				WorkingDir:       "/srv/pihole",
				ExternalNetworks: []string{"proxy", "dns"},
				ComposeBin:       "/usr/bin/podman compose",
				GeneratedAt:      fixedTime,
			},
			goldenFile: "with-networks.service",
			contains: []string{
				"ExecStartPre=/bin/sh -c '/usr/bin/podman network exists proxy || /usr/bin/podman network create proxy'",
				"ExecStartPre=/bin/sh -c '/usr/bin/podman network exists dns || /usr/bin/podman network create dns'",
			},
		},
		{
			name: "custom after values - verify deduplication",
			opts: Opts{
				Name:            "authapp",
				ComposePath:     "/srv/authapp/compose.yml",
				ComposeFilename: "compose.yml",
				WorkingDir:      "/srv/authapp",
				After:           []string{"network-online.target", "pihole.service"},
				ComposeBin:      "/usr/bin/podman compose",
				GeneratedAt:     fixedTime,
			},
			contains: []string{
				"After=network-online.target pihole.service",
				"Requires=network-online.target pihole.service",
			},
			notContain: []string{
				// network-online.target must not appear twice
				"After=network-online.target network-online.target",
			},
		},
		{
			name: "with env files",
			opts: Opts{
				Name:            "webapp",
				ComposePath:     "/srv/webapp/docker-compose.yml",
				ComposeFilename: "docker-compose.yml",
				WorkingDir:      "/srv/webapp",
				EnvFiles:        []string{"/srv/webapp/.env", "/srv/webapp/.env.local"},
				ComposeBin:      "/usr/bin/podman compose",
				GeneratedAt:     fixedTime,
			},
			contains: []string{
				"EnvironmentFile=/srv/webapp/.env",
				"EnvironmentFile=/srv/webapp/.env.local",
			},
		},
		{
			name: "custom timeout",
			opts: Opts{
				Name:            "slowapp",
				ComposePath:     "/srv/slowapp/docker-compose.yml",
				ComposeFilename: "docker-compose.yml",
				WorkingDir:      "/srv/slowapp",
				Timeout:         300,
				ComposeBin:      "/usr/bin/podman compose",
				GeneratedAt:     fixedTime,
			},
			contains: []string{
				"TimeoutStartSec=300",
			},
			notContain: []string{
				"TimeoutStartSec=120",
			},
		},
	}

	// Resolve testdata path relative to this package's location.
	goldenDir := filepath.Join("..", "..", "testdata", "expected")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Generate(tc.opts)

			// Check expected substrings.
			for _, s := range tc.contains {
				if !containsStr(got, s) {
					t.Errorf("output missing expected string: %q\n\nGot:\n%s", s, got)
				}
			}

			// Check not-expected substrings.
			for _, s := range tc.notContain {
				if containsStr(got, s) {
					t.Errorf("output contains unexpected string: %q\n\nGot:\n%s", s, got)
				}
			}

			// Write and compare golden files if specified.
			if tc.goldenFile != "" {
				goldenPath := filepath.Join(goldenDir, tc.goldenFile)

				if os.Getenv("UPDATE_GOLDEN") == "1" {
					if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
						t.Fatalf("failed to create golden dir: %v", err)
					}
					if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
						t.Fatalf("failed to write golden file: %v", err)
					}
					t.Logf("updated golden file: %s", goldenPath)
					return
				}

				expected, err := os.ReadFile(goldenPath)
				if err != nil {
					// Golden file doesn't exist yet; write it and pass.
					if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
						t.Fatalf("failed to create golden dir: %v", err)
					}
					if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
						t.Fatalf("failed to write golden file: %v", err)
					}
					t.Logf("created golden file: %s", goldenPath)
					return
				}

				if got != string(expected) {
					t.Errorf("output does not match golden file %s\n\nGot:\n%s\n\nExpected:\n%s", goldenPath, got, string(expected))
				}
			}
		})
	}
}

func TestUnitFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "myapp", "myapp.service"},
		{"hyphenated", "my-app", "my-app.service"},
		{"with dots", "my.app", "my.app.service"},
		{"empty name", "", ".service"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := UnitFileName(tc.input)
			if got != tc.expected {
				t.Errorf("UnitFileName(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestGenerateDefaults(t *testing.T) {
	// Test that defaults are applied when ComposeBin and Timeout are zero values.
	opts := Opts{
		Name:            "defaults-test",
		ComposePath:     "/srv/defaults/compose.yml",
		ComposeFilename: "compose.yml",
		WorkingDir:      "/srv/defaults",
		GeneratedAt:     fixedTime,
		// ComposeBin intentionally empty
		// Timeout intentionally 0
	}

	got := Generate(opts)

	if !containsStr(got, "TimeoutStartSec=120") {
		t.Errorf("expected default TimeoutStartSec=120, got:\n%s", got)
	}
	if !containsStr(got, "/usr/bin/podman compose") {
		t.Errorf("expected default compose bin, got:\n%s", got)
	}
}

func TestGenerateMultipleEnvFiles(t *testing.T) {
	opts := Opts{
		Name:            "multienv",
		ComposePath:     "/srv/multienv/compose.yml",
		ComposeFilename: "compose.yml",
		WorkingDir:      "/srv/multienv",
		EnvFiles:        []string{"/srv/multienv/.env", "/srv/multienv/.env.db", "/srv/multienv/.env.secrets"},
		ComposeBin:      "/usr/bin/podman compose",
		GeneratedAt:     fixedTime,
	}

	got := Generate(opts)

	for _, envFile := range opts.EnvFiles {
		expected := "EnvironmentFile=" + envFile
		if !containsStr(got, expected) {
			t.Errorf("output missing %q\n\nGot:\n%s", expected, got)
		}
	}
}

func TestGenerateEmptyNetworks(t *testing.T) {
	opts := Opts{
		Name:             "emptynets",
		ComposePath:      "/srv/emptynets/compose.yml",
		ComposeFilename:  "compose.yml",
		WorkingDir:       "/srv/emptynets",
		ExternalNetworks: []string{},
		ComposeBin:       "/usr/bin/podman compose",
		GeneratedAt:      fixedTime,
	}

	got := Generate(opts)

	if containsStr(got, "ExecStartPre=") {
		t.Errorf("output should not contain ExecStartPre for empty networks\n\nGot:\n%s", got)
	}
}

func TestGenerateCustomComposeBin(t *testing.T) {
	opts := Opts{
		Name:             "custom-bin",
		ComposePath:      "/srv/custom/compose.yml",
		ComposeFilename:  "compose.yml",
		WorkingDir:       "/srv/custom",
		ExternalNetworks: []string{"mynet"},
		ComposeBin:       "/usr/local/bin/docker compose",
		GeneratedAt:      fixedTime,
	}

	got := Generate(opts)

	// podmanBin should extract /usr/local/bin/docker from the compose bin
	if !containsStr(got, "/usr/local/bin/docker network exists mynet") {
		t.Errorf("expected podman bin extracted from compose-bin path\n\nGot:\n%s", got)
	}
	if !containsStr(got, "ExecStart=/usr/local/bin/docker compose -f compose.yml up -d") {
		t.Errorf("expected custom compose bin in ExecStart\n\nGot:\n%s", got)
	}
}

func TestGenerateMultipleAfterDeps(t *testing.T) {
	opts := Opts{
		Name:            "multidep",
		ComposePath:     "/srv/multidep/compose.yml",
		ComposeFilename: "compose.yml",
		WorkingDir:      "/srv/multidep",
		After:           []string{"pihole.service", "postgres.service"},
		ComposeBin:      "/usr/bin/podman compose",
		GeneratedAt:     fixedTime,
	}

	got := Generate(opts)

	if !containsStr(got, "network-online.target") {
		t.Errorf("expected network-online.target in After\n\nGot:\n%s", got)
	}
	if !containsStr(got, "pihole.service") {
		t.Errorf("expected pihole.service in After\n\nGot:\n%s", got)
	}
	if !containsStr(got, "postgres.service") {
		t.Errorf("expected postgres.service in After\n\nGot:\n%s", got)
	}
}

func containsStr(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && (haystack == needle || len(haystack) > 0 && stringContains(haystack, needle))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
