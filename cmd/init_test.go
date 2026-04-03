package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bjro/ccbox/internal/config"
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

	// Verify .ccbox.yml exists in the project root (not in .devcontainer/).
	ccboxPath := filepath.Join(dir, config.Filename)
	if _, err := os.Stat(ccboxPath); err != nil {
		t.Errorf("missing %s in project root", config.Filename)
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

func TestInitCommand_CcboxYmlContent(t *testing.T) {
	dir := t.TempDir()
	before := time.Now().UTC()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--stacks", "go,node", "--domains", "api.example.com"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, config.Filename))
	if err != nil {
		t.Fatalf("read %s: %v", config.Filename, err)
	}

	cfg, err := config.Load(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}

	wantStacks := []string{"go", "node"}
	if len(cfg.Stacks) != len(wantStacks) {
		t.Fatalf("Stacks = %v, want %v", cfg.Stacks, wantStacks)
	}
	for i, s := range cfg.Stacks {
		if s != wantStacks[i] {
			t.Errorf("Stacks[%d] = %q, want %q", i, s, wantStacks[i])
		}
	}

	wantDomains := []string{"api.example.com"}
	if len(cfg.ExtraDomains) != len(wantDomains) {
		t.Fatalf("ExtraDomains = %v, want %v", cfg.ExtraDomains, wantDomains)
	}
	for i, d := range cfg.ExtraDomains {
		if d != wantDomains[i] {
			t.Errorf("ExtraDomains[%d] = %q, want %q", i, d, wantDomains[i])
		}
	}

	// GeneratedAt should be recent (within last minute).
	if cfg.GeneratedAt.Before(before) || cfg.GeneratedAt.After(time.Now().UTC()) {
		t.Errorf("GeneratedAt = %v, expected between %v and now", cfg.GeneratedAt, before)
	}

	if cfg.CcboxVersion != version {
		t.Errorf("CcboxVersion = %q, want %q", cfg.CcboxVersion, version)
	}
}

func TestInitCommand_CcboxYmlEmptyDomains(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--stacks", "go"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, config.Filename))
	if err != nil {
		t.Fatalf("read %s: %v", config.Filename, err)
	}

	cfg, err := config.Load(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.ExtraDomains == nil {
		t.Error("ExtraDomains should be non-nil empty slice, got nil")
	}
	if len(cfg.ExtraDomains) != 0 {
		t.Errorf("ExtraDomains = %v, want empty slice", cfg.ExtraDomains)
	}
}
