package cmd

import (
	"github.com/kuyacarlo/quadlet-compose/internal/systemd"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:               "logs <name> [-- extra-flags]",
	Short:             "Show journalctl logs for a managed unit",
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: managedUnitNames,
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		system := getSystem(cmd)

		var extraArgs []string
		if dash := cmd.ArgsLenAtDash(); dash >= 0 {
			extraArgs = args[dash:]
		}
		return systemd.Logs(name, system, extraArgs)
	},
}

func init() {
	logsCmd.Flags().SetInterspersed(false)
	rootCmd.AddCommand(logsCmd)
}
