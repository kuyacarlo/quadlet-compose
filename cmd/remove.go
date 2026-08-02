package cmd

import (
	"fmt"

	"github.com/kuyacarlo/quadlet-compose/internal/systemd"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:               "remove <name>",
	Short:             "Disable, delete, and daemon-reload a managed unit",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: managedUnitNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		system := getSystem(cmd)

		if err := systemd.Remove(name, system); err != nil {
			return err
		}

		fmt.Printf("Removed %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
