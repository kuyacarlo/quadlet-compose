package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kuyacarlo/quadlet-compose/internal/compose"
	"github.com/kuyacarlo/quadlet-compose/internal/unit"
	"github.com/spf13/cobra"
)

var genCmd = &cobra.Command{
	Use:   "gen <compose-file>",
	Short: "Generate a systemd unit from a compose file and print to stdout",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"yml", "yaml"}, cobra.ShellCompDirectiveFilterFileExt
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		content, err := generateUnit(cmd, args[0])
		if err != nil {
			return err
		}
		fmt.Print(content)
		return nil
	},
}

func init() {
	addGenFlags(genCmd)
	rootCmd.AddCommand(genCmd)
}

// addGenFlags adds the shared generation flags to a command.
func addGenFlags(cmd *cobra.Command) {
	cmd.Flags().String("name", "", "override service name (default: compose project name)")
	cmd.Flags().StringSlice("after", nil, "additional After= dependencies (repeatable)")
	cmd.Flags().StringSlice("env-file", nil, "EnvironmentFile= paths (repeatable)")
	cmd.Flags().Int("timeout", 120, "TimeoutStartSec value in seconds")
	cmd.Flags().String("compose-bin", "/usr/bin/podman compose", "compose binary command")
	cmd.Flags().String("in-pod", "", "run containers in a pod (pod name, or 'true' to use project name)")
}

// buildOpts parses the compose file and constructs unit.Opts from flags.
func buildOpts(cmd *cobra.Command, composePath string) (unit.Opts, error) {
	cf, err := compose.Parse(composePath)
	if err != nil {
		return unit.Opts{}, fmt.Errorf("parsing compose file: %w", err)
	}

	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		name = cf.Name
	}

	after, _ := cmd.Flags().GetStringSlice("after")
	envFiles, _ := cmd.Flags().GetStringSlice("env-file")
	timeout, _ := cmd.Flags().GetInt("timeout")
	composeBin, _ := cmd.Flags().GetString("compose-bin")
	inPod, _ := cmd.Flags().GetString("in-pod")

	// If --in-pod=true, use the project name as the pod name.
	if inPod == "true" {
		inPod = name
	}

	return unit.Opts{
		Name:             name,
		ComposePath:      cf.AbsPath,
		ComposeFilename:  cf.Filename,
		WorkingDir:       cf.Dir,
		ExternalNetworks: cf.ExternalNetworks,
		After:            after,
		EnvFiles:         envFiles,
		Timeout:          timeout,
		ComposeBin:       composeBin,
		InPod:            inPod,
		GeneratedAt:      time.Now(),
	}, nil
}

// generateUnit builds opts and generates the unit content string.
func generateUnit(cmd *cobra.Command, composePath string) (string, error) {
	opts, err := buildOpts(cmd, composePath)
	if err != nil {
		return "", err
	}
	return unit.Generate(opts), nil
}

// composeFileCompletion returns a ValidArgsFunction that completes .yml/.yaml files.
func composeFileCompletion() func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"yml", "yaml"}, cobra.ShellCompDirectiveFilterFileExt
	}
}

// managedUnitNames returns unit names for tab completion.
func managedUnitNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Import is circular-free since we only call at completion time.
	system := getSystem(cmd)
	units, err := listManagedNames(system)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var matches []string
	for _, name := range units {
		if strings.HasPrefix(name, toComplete) {
			matches = append(matches, name)
		}
	}
	return matches, cobra.ShellCompDirectiveNoFileComp
}

// listManagedNames returns the names of managed unit files in the unit dir.
func listManagedNames(system bool) ([]string, error) {
	dir, err := unitDirPath(system)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".service") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".service"))
	}
	return names, nil
}

// unitDirPath returns the unit directory without importing systemd (avoids cycle).
func unitDirPath(system bool) (string, error) {
	if system {
		return "/etc/containers/systemd", nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "containers", "systemd"), nil
}
