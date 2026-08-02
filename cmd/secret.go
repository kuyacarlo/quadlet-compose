package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage Podman secrets",
	Long:  "Create, list, and remove Podman secrets used by quadlet containers.",
}

var secretCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a Podman secret",
	Long:  "Create a Podman secret by prompting for a password, generating a random hex string, or reading from a file.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretCreate,
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Podman secrets",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := exec.Command("podman", "secret", "ls")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var secretRmCmd = &cobra.Command{
	Use:     "rm <name>",
	Aliases: []string{"remove"},
	Short:   "Remove a Podman secret",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := exec.Command("podman", "secret", "rm", args[0])
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

func init() {
	secretCreateCmd.Flags().Int("generate", 0, "generate a random hex secret of given byte length")
	secretCreateCmd.Flags().String("from-file", "", "read secret value from file")

	secretCmd.AddCommand(secretCreateCmd)
	secretCmd.AddCommand(secretListCmd)
	secretCmd.AddCommand(secretRmCmd)
	rootCmd.AddCommand(secretCmd)
}

func runSecretCreate(cmd *cobra.Command, args []string) error {
	name := args[0]

	genLen, _ := cmd.Flags().GetInt("generate")
	fromFile, _ := cmd.Flags().GetString("from-file")

	var secret string
	var err error

	switch {
	case genLen > 0:
		secret, err = generateHexSecret(genLen)
		if err != nil {
			return fmt.Errorf("generating secret: %w", err)
		}
	case fromFile != "":
		data, err := os.ReadFile(fromFile)
		if err != nil {
			return fmt.Errorf("reading file: %w", err)
		}
		secret = strings.TrimRight(string(data), "\n")
	default:
		fmt.Fprint(os.Stderr, "Enter secret value: ")
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			return fmt.Errorf("reading password: %w", err)
		}
		secret = string(raw)
	}

	c := exec.Command("podman", "secret", "create", name, "-")
	c.Stdin = strings.NewReader(secret)
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("podman secret create: %w", err)
	}

	fmt.Fprintf(os.Stdout, "✓ Created secret: %s\n", name)
	return nil
}

// generateHexSecret returns a hex-encoded string from n random bytes.
func generateHexSecret(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
