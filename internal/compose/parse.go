// Package compose parses Docker Compose files into a minimal representation
// used by quadlet-compose for generating systemd quadlet units.
package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"go.yaml.in/yaml/v3"
)

// ComposeFile holds the parsed metadata from a compose YAML file.
type ComposeFile struct {
	Name             string   // Top-level name field, or derived from parent dir.
	ExternalNetworks []string // Sorted list of networks marked external: true.
	AbsPath          string   // Absolute path to the compose file.
	Dir              string   // Directory containing the compose file.
	Filename         string   // Base filename.
}

// rawCompose is the intermediate YAML structure we decode into.
type rawCompose struct {
	Name     string                        `yaml:"name"`
	Networks map[string]rawNetworkDefinition `yaml:"networks"`
}

// rawNetworkDefinition handles both forms:
//
//	external: true
//	external:
//	  name: custom
type rawNetworkDefinition struct {
	External rawExternal `yaml:"external"`
}

// rawExternal handles the polymorphic external field.
// It can be a bare bool (`external: true`) or a mapping (`external:\n  name: foo`).
type rawExternal struct {
	IsExternal bool
	Name       string
}

// UnmarshalYAML implements custom unmarshalling for the external field.
func (e *rawExternal) UnmarshalYAML(value *yaml.Node) error {
	// Case 1: external: true/false (scalar bool)
	if value.Kind == yaml.ScalarNode {
		var b bool
		if err := value.Decode(&b); err != nil {
			return err
		}
		e.IsExternal = b
		return nil
	}

	// Case 2: external:\n  name: custom (mapping)
	if value.Kind == yaml.MappingNode {
		var m struct {
			Name string `yaml:"name"`
		}
		if err := value.Decode(&m); err != nil {
			return err
		}
		e.IsExternal = true
		e.Name = m.Name
		return nil
	}

	return fmt.Errorf("unexpected external field type: %v", value.Kind)
}

// Parse reads and parses a compose YAML file at the given path.
// It extracts the project name and any external networks.
// If the top-level name field is empty, the parent directory name is used.
// ExternalNetworks is always returned sorted.
func Parse(path string) (*ComposeFile, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("reading compose file: %w", err)
	}

	var raw rawCompose
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing compose YAML: %w", err)
	}

	name := raw.Name
	if name == "" {
		name = filepath.Base(filepath.Dir(absPath))
	}

	var extNets []string
	for netName, netDef := range raw.Networks {
		if netDef.External.IsExternal {
			extNets = append(extNets, netName)
		}
	}
	sort.Strings(extNets)

	return &ComposeFile{
		Name:             name,
		ExternalNetworks: extNets,
		AbsPath:          absPath,
		Dir:              filepath.Dir(absPath),
		Filename:         filepath.Base(absPath),
	}, nil
}
