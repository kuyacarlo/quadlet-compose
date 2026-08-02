package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionsCmd = &cobra.Command{
	Use:       "completions <shell>",
	Short:     "Generate shell completions",
	Long:      "Generate shell completion scripts for bash, zsh, or fish.",
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		default:
			return fmt.Errorf("unsupported shell: %s (use bash, zsh, or fish)", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionsCmd)
}
