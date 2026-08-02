package cmd

import (
	"github.com/kuyacarlo/quadlet-compose/internal/systemd"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:               "status <name>",
	Short:             "Show systemctl status for a managed unit",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: managedUnitNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		system := getSystem(cmd)
		return systemd.Status(name, system)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
