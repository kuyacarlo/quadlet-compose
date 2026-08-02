package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestGenCommand_SimpleFile(t *testing.T) {
	testdata := resolveTestdata(t)
	composePath := filepath.Join(testdata, "simple.yml")

	out, err := executeGen(t, composePath)
	if err != nil {
		t.Fatalf("gen command failed: %v", err)
	}

	assertContains(t, out, "# Managed by quadlet-compose")
	assertContains(t, out, "Description=quadlet-compose: myapp")
	assertContains(t, out, "ExecStart=/usr/bin/podman compose -f simple.yml up -d")
	assertContains(t, out, "WantedBy=default.target")
}

func TestGenCommand_WithNetworks(t *testing.T) {
	testdata := resolveTestdata(t)
	composePath := filepath.Join(testdata, "with-networks.yml")

	out, err := executeGen(t, composePath)
	if err != nil {
		t.Fatalf("gen command failed: %v", err)
	}

	assertContains(t, out, "Description=quadlet-compose: mystack")
	assertContains(t, out, "ExecStartPre=/bin/sh -c '/usr/bin/podman network exists backend || /usr/bin/podman network create backend'")
	assertContains(t, out, "ExecStartPre=/bin/sh -c '/usr/bin/podman network exists frontend || /usr/bin/podman network create frontend'")
}

func TestGenCommand_WithNameFlag(t *testing.T) {
	testdata := resolveTestdata(t)
	composePath := filepath.Join(testdata, "simple.yml")

	out, err := executeGenWithArgs(t, composePath, "--name", "custom-name")
	if err != nil {
		t.Fatalf("gen command failed: %v", err)
	}

	assertContains(t, out, "Description=quadlet-compose: custom-name")
}

func TestGenCommand_WithAfterFlag(t *testing.T) {
	testdata := resolveTestdata(t)
	composePath := filepath.Join(testdata, "simple.yml")

	out, err := executeGenWithArgs(t, composePath, "--after", "pihole.service")
	if err != nil {
		t.Fatalf("gen command failed: %v", err)
	}

	assertContains(t, out, "pihole.service")
}

func TestGenCommand_WithEnvFileFlag(t *testing.T) {
	testdata := resolveTestdata(t)
	composePath := filepath.Join(testdata, "simple.yml")

	out, err := executeGenWithArgs(t, composePath, "--env-file", "/srv/app/.env")
	if err != nil {
		t.Fatalf("gen command failed: %v", err)
	}

	assertContains(t, out, "EnvironmentFile=/srv/app/.env")
}

func TestGenCommand_WithTimeoutFlag(t *testing.T) {
	testdata := resolveTestdata(t)
	composePath := filepath.Join(testdata, "simple.yml")

	out, err := executeGenWithArgs(t, composePath, "--timeout", "300")
	if err != nil {
		t.Fatalf("gen command failed: %v", err)
	}

	assertContains(t, out, "TimeoutStartSec=300")
}

func TestGenCommand_MissingFile(t *testing.T) {
	_, err := executeGen(t, "/nonexistent/path/compose.yml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestGenCommand_NoArgs(t *testing.T) {
	_, errNoArgs := captureStdout(func() error {
		rootCmd.SetArgs([]string{"gen"})
		return rootCmd.Execute()
	})
	if errNoArgs == nil {
		t.Fatal("expected error for no args, got nil")
	}
}

// --- helpers ---

// stdoutMu serializes stdout capture since os.Stdout is global.
var stdoutMu sync.Mutex

func resolveTestdata(t *testing.T) string {
	t.Helper()
	// cmd/ is one level below root, testdata is at root
	td, err := filepath.Abs("../testdata")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(td); err != nil {
		t.Fatalf("testdata not found at %s: %v", td, err)
	}
	return td
}

func executeGen(t *testing.T, composePath string) (string, error) {
	t.Helper()
	return executeGenWithArgs(t, composePath)
}

func executeGenWithArgs(t *testing.T, composePath string, extraArgs ...string) (string, error) {
	t.Helper()
	args := []string{"gen", composePath}
	args = append(args, extraArgs...)

	var execErr error
	out, _ := captureStdout(func() error {
		rootCmd.SetArgs(args)
		execErr = rootCmd.Execute()
		return execErr
	})
	return out, execErr
}

func captureStdout(f func() error) (string, error) {
	stdoutMu.Lock()
	defer stdoutMu.Unlock()

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		io.Copy(&buf, r)
		close(done)
	}()

	execErr := f()

	w.Close()
	<-done
	os.Stdout = origStdout

	return buf.String(), execErr
}

func assertContains(t *testing.T, output, substr string) {
	t.Helper()
	if !strings.Contains(output, substr) {
		t.Errorf("output missing %q\n\nGot:\n%s", substr, output)
	}
}
