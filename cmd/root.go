package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "quadlet-compose",
	Short: "Convert compose files to Podman quadlet units",
	Long:  "quadlet-compose (alias: complet) converts Docker Compose files into systemd units managed by Podman, handling install, enable, disable, and lifecycle operations.",
	Aliases: []string{"complet"},
}

func init() {
	// If invoked as "complet", override Use so help text looks right.
	cobra.OnInitialize(func() {
		bin := filepath.Base(os.Args[0])
		if bin == "complet" {
			rootCmd.Use = "complet"
		}
	})
	rootCmd.PersistentFlags().Bool("user", true, "operate in user mode (systemctl --user)")
	rootCmd.PersistentFlags().Bool("system", false, "operate in system mode")
}

// SetVersion sets the version string displayed by --version.
func SetVersion(v string) {
	rootCmd.Version = v
}

// Execute runs the root command. Called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// getSystem returns whether system mode is active.
// --system=true takes precedence; otherwise it's the inverse of --user.
func getSystem(cmd *cobra.Command) bool {
	systemFlag, _ := cmd.Flags().GetBool("system")
	if systemFlag {
		return true
	}
	userFlag, _ := cmd.Flags().GetBool("user")
	return !userFlag
}
