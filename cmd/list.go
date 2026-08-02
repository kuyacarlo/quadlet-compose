package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kuyacarlo/quadlet-compose/internal/systemd"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed units",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		system := getSystem(cmd)

		units, err := systemd.List(system)
		if err != nil {
			return err
		}

		if len(units) == 0 {
			fmt.Println("No managed units found.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSOURCE\tSTATUS")
		for _, u := range units {
			fmt.Fprintf(w, "%s\t%s\t%s\n", u.Name, u.Source, u.Status)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
