package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/kuyacarlo/quadlet-compose/internal/systemd"
	"github.com/kuyacarlo/quadlet-compose/internal/unit"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install <compose-file>",
	Short: "Generate and install a systemd unit, then daemon-reload",
	Args:  cobra.ExactArgs(1),
	ValidArgsFunction: composeFileCompletion(),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts, err := buildOpts(cmd, args[0])
		if err != nil {
			return err
		}

		content := unit.Generate(opts)
		system := getSystem(cmd)

		if err := systemd.Install(content, opts.Name, system); err != nil {
			return err
		}

		dir, _ := systemd.UnitDir(system)
		path := filepath.Join(dir, unit.UnitFileName(opts.Name))
		fmt.Printf("Installed %s → %s\n", opts.Name, path)
		return nil
	},
}

func init() {
	addGenFlags(installCmd)
	rootCmd.AddCommand(installCmd)
}
