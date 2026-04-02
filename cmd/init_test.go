package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitCommand_GeneratesDevcontainer(t *testing.T) {
	// Create a temp dir with a go.mod to trigger Go stack detection.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Change to temp dir so init detects the Go stack.
	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify key files were generated.
	expected := []string{
		"Dockerfile",
		"devcontainer.json",
		"init-firewall.sh",
		"warmup-dns.sh",
		"dynamic-domains.conf",
		"claude-user-settings.json",
		"sync-claude-settings.sh",
		"README.md",
	}

	devcontainerDir := filepath.Join(dir, ".devcontainer")
	for _, name := range expected {
		path := filepath.Join(devcontainerDir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("missing file: %s", name)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("empty file: %s", name)
		}
	}
}

func TestInitCommand_WithStacksFlag(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--stacks", "go,node"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify Dockerfile was generated.
	path := filepath.Join(dir, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(path); err != nil {
		t.Error("missing Dockerfile")
	}
}
