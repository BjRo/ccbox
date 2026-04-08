//go:build integration

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bjro/agentbox/internal/config"
)

func TestIntegration_UpdatePreservesCustomizations(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Init with Go stack.
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--dir", dir, "--stack", "go"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	devDir := filepath.Join(dir, ".devcontainer")

	// Add custom content to the custom stage.
	dfPath := filepath.Join(devDir, "Dockerfile")
	original := readFile(t, dfPath)
	modified := strings.Replace(original, "FROM agentbox AS custom", "FROM agentbox AS custom\nRUN echo custom-tool", 1)
	if err := os.WriteFile(dfPath, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}

	// Edit mise-config.toml to set custom versions.
	configPath := filepath.Join(devDir, "mise-config.toml")
	customConfig := "[tools]\ngo = \"1.23\"\nnode = \"22\"\n"
	if err := os.WriteFile(configPath, []byte(customConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run update.
	updateCmd := newRootCmd(nil)
	updateCmd.SetArgs([]string{"update", "--dir", dir})
	if err := updateCmd.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify custom RUN line preserved.
	updated := readFile(t, dfPath)
	if !strings.Contains(updated, "RUN echo custom-tool") {
		t.Error("custom stage content should be preserved after update")
	}

	// Verify mise-config.toml preserved.
	configContent := readFile(t, configPath)
	if configContent != customConfig {
		t.Errorf("mise-config.toml should be preserved; got %q", configContent)
	}

	// Verify agentbox stage is freshly rendered.
	if !strings.Contains(updated, "FROM debian:bookworm-slim AS agentbox") {
		t.Error("Dockerfile should contain fresh agentbox stage")
	}

	// Verify whitespace between stages: WORKDIR /workspace\n\nFROM agentbox AS custom
	if !strings.Contains(updated, "WORKDIR /workspace\n\nFROM agentbox AS custom") {
		t.Error("Dockerfile should have exactly one blank line between agentbox stage and custom stage")
	}
}

func TestIntegration_UpdateWithStackChange(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Init with Go stack.
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--dir", dir, "--stack", "go"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Update with Go + Node.
	updateCmd := newRootCmd(nil)
	updateCmd.SetArgs([]string{"update", "--dir", dir, "--stack", "go,node"})
	if err := updateCmd.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify node LSP appears in Dockerfile.
	devDir := filepath.Join(dir, ".devcontainer")
	dockerfile := readFile(t, filepath.Join(devDir, "Dockerfile"))
	if !strings.Contains(dockerfile, "typescript-language-server") {
		t.Error("Dockerfile should contain typescript-language-server after adding node stack")
	}

	// Verify .agentbox.yml has both stacks.
	f, err := os.Open(filepath.Join(dir, config.Filename))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := config.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Stacks) != 2 {
		t.Errorf("expected 2 stacks, got %d: %v", len(cfg.Stacks), cfg.Stacks)
	}
}

func TestIntegration_UpdateForceMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Init with Go stack.
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--dir", dir, "--stack", "go"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Remove the custom stage from the Dockerfile.
	devDir := filepath.Join(dir, ".devcontainer")
	dfPath := filepath.Join(devDir, "Dockerfile")
	original := readFile(t, dfPath)
	// Keep only the agentbox stage (everything before FROM agentbox AS custom).
	idx := strings.Index(original, "FROM agentbox AS custom")
	if idx == -1 {
		t.Fatal("initial Dockerfile should contain custom stage")
	}
	truncated := original[:idx]
	if err := os.WriteFile(dfPath, []byte(truncated), 0o644); err != nil {
		t.Fatal(err)
	}

	// Update without --force should fail.
	updateCmd := newRootCmd(nil)
	updateCmd.SetArgs([]string{"update", "--dir", dir})
	err := updateCmd.Execute()
	if err == nil {
		t.Fatal("expected error without --force")
	}

	// Update with --force should succeed.
	forceCmd := newRootCmd(nil)
	forceCmd.SetArgs([]string{"update", "--dir", dir, "--force"})
	if err := forceCmd.Execute(); err != nil {
		t.Fatalf("update --force: %v", err)
	}

	// Verify fresh custom stage stub.
	updated := readFile(t, dfPath)
	if !strings.Contains(updated, "FROM agentbox AS custom") {
		t.Error("Dockerfile should contain FROM agentbox AS custom after --force")
	}
	if !strings.Contains(updated, "USER CUSTOMIZATIONS") {
		t.Error("Dockerfile should contain custom stage comments after --force")
	}
}

func TestIntegration_UpdateIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Init with Go stack.
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--dir", dir, "--stack", "go"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	devDir := filepath.Join(dir, ".devcontainer")

	// Read all files after init.
	beforeFiles := make(map[string]string)
	for _, name := range expectedFiles {
		beforeFiles[name] = readFile(t, filepath.Join(devDir, name))
	}

	// Run update with same config.
	updateCmd := newRootCmd(nil)
	updateCmd.SetArgs([]string{"update", "--dir", dir})
	if err := updateCmd.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify all files are identical (except .agentbox.yml which has a new timestamp).
	for _, name := range expectedFiles {
		after := readFile(t, filepath.Join(devDir, name))
		if after != beforeFiles[name] {
			t.Errorf("file %s changed after idempotent update", name)
		}
	}
}
