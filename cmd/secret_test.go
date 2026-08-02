package cmd

import (
	"encoding/hex"
	"fmt"
	"testing"
)

func TestGenerateHexSecret_Length(t *testing.T) {
	tests := []struct {
		byteLen    int
		wantHexLen int
	}{
		{16, 32},
		{32, 64},
		{1, 2},
		{64, 128},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d_bytes", tt.byteLen), func(t *testing.T) {
			result, err := generateHexSecret(tt.byteLen)
			if err != nil {
				t.Fatalf("generateHexSecret(%d) returned error: %v", tt.byteLen, err)
			}
			if len(result) != tt.wantHexLen {
				t.Errorf("generateHexSecret(%d) = %q (len %d), want len %d", tt.byteLen, result, len(result), tt.wantHexLen)
			}
			// Verify it's valid hex
			decoded, err := hex.DecodeString(result)
			if err != nil {
				t.Errorf("generateHexSecret(%d) produced invalid hex: %v", tt.byteLen, err)
			}
			if len(decoded) != tt.byteLen {
				t.Errorf("decoded length = %d, want %d", len(decoded), tt.byteLen)
			}
		})
	}
}

func TestGenerateHexSecret_Unique(t *testing.T) {
	a, _ := generateHexSecret(32)
	b, _ := generateHexSecret(32)
	if a == b {
		t.Error("two calls to generateHexSecret returned the same value")
	}
}

func TestSecretCommandStructure(t *testing.T) {
	// Verify secretCmd is registered on rootCmd
	found := false
	for _, c := range rootCmd.Commands() {
		if c.Use == "secret" {
			found = true

			// Check subcommands
			subNames := map[string]bool{}
			for _, sub := range c.Commands() {
				subNames[sub.Use] = true
			}

			if !subNames["create <name>"] {
				t.Error("secret command missing 'create' subcommand")
			}
			if !subNames["list"] {
				t.Error("secret command missing 'list' subcommand")
			}
			if !subNames["rm <name>"] {
				t.Error("secret command missing 'rm' subcommand")
			}

			// Verify rm has alias
			for _, sub := range c.Commands() {
				if sub.Use == "rm <name>" {
					hasAlias := false
					for _, a := range sub.Aliases {
						if a == "remove" {
							hasAlias = true
						}
					}
					if !hasAlias {
						t.Error("rm subcommand missing 'remove' alias")
					}
				}
			}

			break
		}
	}
	if !found {
		t.Error("'secret' command not registered on root")
	}
}

func TestSecretCreateFlags(t *testing.T) {
	// Verify flags exist on create subcommand
	for _, c := range secretCmd.Commands() {
		if c.Use == "create <name>" {
			f := c.Flags().Lookup("generate")
			if f == nil {
				t.Error("create subcommand missing --generate flag")
			}
			f = c.Flags().Lookup("from-file")
			if f == nil {
				t.Error("create subcommand missing --from-file flag")
			}
			return
		}
	}
	t.Error("create subcommand not found")
}
