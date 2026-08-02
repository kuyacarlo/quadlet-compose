package cmd

import (
	"fmt"

	"github.com/kuyacarlo/quadlet-compose/internal/systemd"
	"github.com/spf13/cobra"
)

var disableCmd = &cobra.Command{
	Use:               "disable <name>",
	Short:             "Disable and stop a managed unit",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: managedUnitNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		system := getSystem(cmd)

		if err := systemd.Disable(name, system); err != nil {
			return err
		}

		fmt.Printf("Disabled %s\n", name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(disableCmd)
}
