package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bjro/agentbox/internal/config"
)

// seedInitDir runs agentbox init in dir with the given stacks, returning the
// .devcontainer path. It calls t.Fatal on error.
func seedInitDir(t *testing.T, dir string, stacks string) string {
	t.Helper()
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--dir", dir, "--stack", stacks})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("seed init: %v", err)
	}
	return filepath.Join(dir, ".devcontainer")
}

func TestUpdateCommand_RequiresAgentboxYml(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"update", "--dir", dir})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when .agentbox.yml is missing")
	}
	if !strings.Contains(err.Error(), "agentbox init") {
		t.Errorf("error should mention 'agentbox init'; got: %s", err.Error())
	}
}

func TestUpdateCommand_RequiresDockerfile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create .agentbox.yml but no .devcontainer/Dockerfile.
	cfg := config.Config{
		Version: 1,
		Stacks:  []string{"go"},
	}
	writeCfg(t, filepath.Join(dir, config.Filename), cfg)

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"update", "--dir", dir})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when Dockerfile is missing")
	}
	if !strings.Contains(err.Error(), "Dockerfile") {
		t.Errorf("error should mention 'Dockerfile'; got: %s", err.Error())
	}
}

func TestUpdateCommand_RequiresCustomStage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create .agentbox.yml and a Dockerfile without custom stage.
	cfg := config.Config{
		Version: 1,
		Stacks:  []string{"go"},
	}
	writeCfg(t, filepath.Join(dir, config.Filename), cfg)

	outDir := filepath.Join(dir, ".devcontainer")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "Dockerfile"), []byte("FROM debian:bookworm-slim\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"update", "--dir", dir})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when custom stage is missing")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should mention '--force'; got: %s", err.Error())
	}
}

func TestUpdateCommand_ForceRegeneration(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create .agentbox.yml and a Dockerfile without custom stage.
	cfg := config.Config{
		Version: 1,
		Stacks:  []string{"go"},
	}
	writeCfg(t, filepath.Join(dir, config.Filename), cfg)

	outDir := filepath.Join(dir, ".devcontainer")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "Dockerfile"), []byte("FROM debian:bookworm-slim\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"update", "--dir", dir, "--force"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("update --force: %v", err)
	}

	// Verify fresh custom stage stub was generated.
	dockerfile, err := os.ReadFile(filepath.Join(outDir, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerfile), "FROM agentbox AS custom") {
		t.Error("Dockerfile should contain FROM agentbox AS custom after --force")
	}

	// Verify codex files are produced during force regeneration.
	codexConfig, err := os.ReadFile(filepath.Join(outDir, "codex-config.toml"))
	if err != nil {
		t.Fatalf("read codex-config.toml: %v", err)
	}
	if len(codexConfig) == 0 {
		t.Error("codex-config.toml should be non-empty after --force")
	}
	syncCodex, err := os.ReadFile(filepath.Join(outDir, "sync-codex-settings.sh"))
	if err != nil {
		t.Fatalf("read sync-codex-settings.sh: %v", err)
	}
	if len(syncCodex) == 0 {
		t.Error("sync-codex-settings.sh should be non-empty after --force")
	}
}

func TestUpdateCommand_PreservesCustomStage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	devDir := seedInitDir(t, dir, "go")

	// Add custom content to the custom stage.
	dfPath := filepath.Join(devDir, "Dockerfile")
	original, err := os.ReadFile(dfPath)
	if err != nil {
		t.Fatal(err)
	}
	modified := strings.Replace(string(original), "FROM agentbox AS custom", "FROM agentbox AS custom\nRUN echo hello-custom", 1)
	if err := os.WriteFile(dfPath, []byte(modified), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run update.
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"update", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify custom content is preserved.
	updated, err := os.ReadFile(dfPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "RUN echo hello-custom") {
		t.Error("custom stage content should be preserved after update")
	}
}

func TestUpdateCommand_PreservesMiseConfigToml(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	devDir := seedInitDir(t, dir, "go")

	// Modify mise-config.toml.
	configPath := filepath.Join(devDir, "mise-config.toml")
	customConfig := "[tools]\ngo = \"1.23\"\nnode = \"22\"\n"
	if err := os.WriteFile(configPath, []byte(customConfig), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run update.
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"update", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify mise-config.toml is preserved.
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != customConfig {
		t.Errorf("mise-config.toml should be preserved; got %q, want %q", string(content), customConfig)
	}
}

func TestUpdateCommand_StackFlagOverrides(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	seedInitDir(t, dir, "go")

	// Update with different stacks.
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"update", "--dir", dir, "--stack", "go,node"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify .agentbox.yml has updated stacks.
	f, err := os.Open(filepath.Join(dir, config.Filename))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	cfg, err := config.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Stacks) != 2 {
		t.Errorf("expected 2 stacks, got %d: %v", len(cfg.Stacks), cfg.Stacks)
	}

	// Verify Dockerfile has node LSP.
	devDir := filepath.Join(dir, ".devcontainer")
	dockerfile, err := os.ReadFile(filepath.Join(devDir, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(dockerfile), "typescript-language-server") {
		t.Error("Dockerfile should contain typescript-language-server after adding node stack")
	}
}

func TestUpdateCommand_ExtraDomainsFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	seedInitDir(t, dir, "go")

	// Update with extra domains.
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"update", "--dir", dir, "--extra-domains", "api.example.com"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify .agentbox.yml has extra domains.
	f, err := os.Open(filepath.Join(dir, config.Filename))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	cfg, err := config.Load(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.ExtraDomains) != 1 || cfg.ExtraDomains[0] != "api.example.com" {
		t.Errorf("expected extra_domains [api.example.com], got %v", cfg.ExtraDomains)
	}
}

func TestUpdateCommand_NoRuntimeVersionFlag(t *testing.T) {
	t.Parallel()
	cmd := newRootCmd(nil)
	updateCmd, _, err := cmd.Find([]string{"update"})
	if err != nil {
		t.Fatalf("find update command: %v", err)
	}

	flag := updateCmd.Flags().Lookup("runtime-version")
	if flag != nil {
		t.Error("update command should not have --runtime-version flag")
	}
}

func TestUpdateCommand_RegeneratesAgentboxStage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	devDir := seedInitDir(t, dir, "go")

	// Verify initial Dockerfile has gopls but not pyright.
	initial, err := os.ReadFile(filepath.Join(devDir, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(initial), "gopls") {
		t.Fatal("initial Dockerfile should contain gopls")
	}
	if strings.Contains(string(initial), "pyright") {
		t.Fatal("initial Dockerfile should not contain pyright")
	}

	// Update to include python stack.
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"update", "--dir", dir, "--stack", "go,python"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("update: %v", err)
	}

	// Verify regenerated Dockerfile has pyright.
	updated, err := os.ReadFile(filepath.Join(devDir, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "pyright") {
		t.Error("updated Dockerfile should contain pyright after adding python stack")
	}
}

// writeCfg writes a config to the given path.
func writeCfg(t *testing.T, path string, cfg config.Config) {
	t.Helper()
	var buf strings.Builder
	if err := config.Write(&buf, cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(path, []byte(buf.String()), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}
