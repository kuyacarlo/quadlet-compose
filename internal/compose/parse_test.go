package compose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	// Resolve testdata relative to the project root.
	// Tests run from the package directory, so we go up two levels.
	testdata, err := filepath.Abs("../../testdata")
	if err != nil {
		t.Fatal(err)
	}

	// Create a temp dir with an invalid YAML file for the error case.
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.yml")
	if err := os.WriteFile(invalidPath, []byte(":\n  :\n[broken"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file with no name field to test directory fallback.
	fallbackDir := filepath.Join(tmpDir, "my-project")
	if err := os.MkdirAll(fallbackDir, 0755); err != nil {
		t.Fatal(err)
	}
	fallbackPath := filepath.Join(fallbackDir, "compose.yml")
	if err := os.WriteFile(fallbackPath, []byte("services:\n  web:\n    image: nginx\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// external: with name subfield (mapping form)
	externalNameDir := filepath.Join(tmpDir, "ext-name-proj")
	if err := os.MkdirAll(externalNameDir, 0755); err != nil {
		t.Fatal(err)
	}
	externalNamePath := filepath.Join(externalNameDir, "compose.yml")
	if err := os.WriteFile(externalNamePath, []byte(`name: extname
networks:
  mynet:
    external:
      name: custom-net
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Network with external: false (should not be included)
	externalFalseDir := filepath.Join(tmpDir, "ext-false-proj")
	if err := os.MkdirAll(externalFalseDir, 0755); err != nil {
		t.Fatal(err)
	}
	externalFalsePath := filepath.Join(externalFalseDir, "compose.yml")
	if err := os.WriteFile(externalFalsePath, []byte(`name: noext
networks:
  internal:
    external: false
  outside:
    external: true
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Empty networks map
	emptyNetsDir := filepath.Join(tmpDir, "empty-nets-proj")
	if err := os.MkdirAll(emptyNetsDir, 0755); err != nil {
		t.Fatal(err)
	}
	emptyNetsPath := filepath.Join(emptyNetsDir, "compose.yml")
	if err := os.WriteFile(emptyNetsPath, []byte(`name: emptynets
networks: {}
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Minimal file: completely empty YAML
	emptyYAMLDir := filepath.Join(tmpDir, "empty-yaml-proj")
	if err := os.MkdirAll(emptyYAMLDir, 0755); err != nil {
		t.Fatal(err)
	}
	emptyYAMLPath := filepath.Join(emptyYAMLDir, "compose.yml")
	if err := os.WriteFile(emptyYAMLPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		path       string
		wantName   string
		wantNets   []string
		wantErr    bool
	}{
		{
			name:     "simple file, no networks",
			path:     filepath.Join(testdata, "simple.yml"),
			wantName: "myapp",
			wantNets: nil,
		},
		{
			name:     "file with external networks, sorted",
			path:     filepath.Join(testdata, "with-networks.yml"),
			wantName: "mystack",
			wantNets: []string{"backend", "frontend"},
		},
		{
			name:    "file not found",
			path:    filepath.Join(testdata, "nonexistent.yml"),
			wantErr: true,
		},
		{
			name:    "invalid YAML",
			path:    invalidPath,
			wantErr: true,
		},
		{
			name:     "name fallback to directory",
			path:     fallbackPath,
			wantName: "my-project",
			wantNets: nil,
		},
		{
			name:     "external with name mapping",
			path:     externalNamePath,
			wantName: "extname",
			wantNets: []string{"mynet"},
		},
		{
			name:     "external false is excluded",
			path:     externalFalsePath,
			wantName: "noext",
			wantNets: []string{"outside"},
		},
		{
			name:     "empty networks map",
			path:     emptyNetsPath,
			wantName: "emptynets",
			wantNets: nil,
		},
		{
			name:     "empty YAML file uses dir name",
			path:     emptyYAMLPath,
			wantName: "empty-yaml-proj",
			wantNets: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := Parse(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cf.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", cf.Name, tt.wantName)
			}

			if !sliceEqual(cf.ExternalNetworks, tt.wantNets) {
				t.Errorf("ExternalNetworks = %v, want %v", cf.ExternalNetworks, tt.wantNets)
			}

			// Verify structural fields are populated.
			if cf.AbsPath == "" {
				t.Error("AbsPath is empty")
			}
			if cf.Dir == "" {
				t.Error("Dir is empty")
			}
			if cf.Filename == "" {
				t.Error("Filename is empty")
			}
		})
	}
}

func TestParseAbsPathFields(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "compose.yml")
	if err := os.WriteFile(composePath, []byte("name: fieldtest\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cf, err := Parse(composePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cf.AbsPath != composePath {
		t.Errorf("AbsPath = %q, want %q", cf.AbsPath, composePath)
	}
	if cf.Dir != tmpDir {
		t.Errorf("Dir = %q, want %q", cf.Dir, tmpDir)
	}
	if cf.Filename != "compose.yml" {
		t.Errorf("Filename = %q, want %q", cf.Filename, "compose.yml")
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
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
