//go:build integration

package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bjro/agentbox/internal/config"
)

// assertFileExists stats the file at path and returns its os.FileInfo.
// It calls t.Fatalf if the file does not exist.
func assertFileExists(t *testing.T, path string) os.FileInfo {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("expected file to exist: %s", path)
	}
	return info
}

// readFile reads the file at path and returns its content as a string.
// It calls t.Fatalf on error.
func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(data)
}

// expectedFiles lists the 8 files that agentbox init generates inside .devcontainer/.
// Intentionally coupled with the file map in cmd/init.go's RunE -- update both together.
var expectedFiles = []string{
	"Dockerfile",
	"devcontainer.json",
	"init-firewall.sh",
	"warmup-dns.sh",
	"dynamic-domains.conf",
	"claude-user-settings.json",
	"sync-claude-settings.sh",
	"README.md",
}

// executableScripts lists the shell scripts that must have the executable bit set.
// Intentionally coupled with the chmod list in cmd/init.go's RunE -- update both together.
var executableScripts = []string{
	"init-firewall.sh",
	"warmup-dns.sh",
	"sync-claude-settings.sh",
}

func TestIntegration_SingleGoStack(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--dir", dir, "--non-interactive"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	devcontainerDir := filepath.Join(dir, ".devcontainer")

	// All 8 files exist and are non-empty.
	for _, name := range expectedFiles {
		info := assertFileExists(t, filepath.Join(devcontainerDir, name))
		if info.Size() == 0 {
			t.Errorf("file is empty: %s", name)
		}
	}

	// Dockerfile content assertions.
	dockerfile := readFile(t, filepath.Join(devcontainerDir, "Dockerfile"))
	if !strings.Contains(dockerfile, `go = "latest"`) {
		t.Error("Dockerfile should contain go runtime in mise config")
	}
	if !strings.Contains(dockerfile, "go install golang.org/x/tools/gopls@latest") {
		t.Error("Dockerfile should contain gopls install command")
	}
	// Negative: no other runtimes.
	for _, absent := range []string{"python", "ruby", "rust"} {
		if strings.Contains(dockerfile, absent+` = "`) {
			t.Errorf("Dockerfile should not contain %s runtime entry", absent)
		}
	}

	// devcontainer.json: valid JSON with expected fields.
	// Unmarshal validates JSON syntax; values are checked via string assertions below.
	devcontainer := readFile(t, filepath.Join(devcontainerDir, "devcontainer.json"))
	var devcontainerMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(devcontainer), &devcontainerMap); err != nil {
		t.Fatalf("devcontainer.json is not valid JSON: %v", err)
	}
	if !strings.Contains(devcontainer, `"dockerfile": "Dockerfile"`) {
		t.Error("devcontainer.json should reference Dockerfile")
	}

	// init-firewall.sh: contains AlwaysOn static domains in dig section.
	initFirewall := readFile(t, filepath.Join(devcontainerDir, "init-firewall.sh"))
	for _, domain := range []string{"api.github.com", "github.com"} {
		digLine := "dig +short '" + domain + "'"
		if !strings.Contains(initFirewall, digLine) {
			t.Errorf("init-firewall.sh should contain dig resolution for %s", domain)
		}
	}
	// proxy.golang.org is Dynamic, so it should NOT appear in the static dig section.
	if strings.Contains(initFirewall, "dig +short 'proxy.golang.org'") {
		t.Error("init-firewall.sh should not contain dig resolution for proxy.golang.org (it is Dynamic)")
	}

	// Shell script permissions.
	for _, name := range executableScripts {
		info := assertFileExists(t, filepath.Join(devcontainerDir, name))
		if info.Mode().Perm()&0o111 == 0 {
			t.Errorf("%s should be executable", name)
		}
	}

	// dynamic-domains.conf: contains Go dynamic domains.
	dynamicDomains := readFile(t, filepath.Join(devcontainerDir, "dynamic-domains.conf"))
	if !strings.Contains(dynamicDomains, "proxy.golang.org") {
		t.Error("dynamic-domains.conf should contain proxy.golang.org")
	}

	// claude-user-settings.json: valid JSON with gopls plugin.
	// Unmarshal validates JSON syntax; values are checked via string assertions below.
	claudeSettings := readFile(t, filepath.Join(devcontainerDir, "claude-user-settings.json"))
	var claudeMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(claudeSettings), &claudeMap); err != nil {
		t.Fatalf("claude-user-settings.json is not valid JSON: %v", err)
	}
	if !strings.Contains(claudeSettings, `"gopls"`) {
		t.Error("claude-user-settings.json should contain gopls plugin")
	}
	if strings.Contains(claudeSettings, `"typescript"`) {
		t.Error("claude-user-settings.json should not contain typescript plugin (single Go stack)")
	}

	// README.md: contains Go stack listing.
	readme := readFile(t, filepath.Join(devcontainerDir, "README.md"))
	if !strings.Contains(readme, "- go\n") {
		t.Error("README.md should list go as a detected stack")
	}
}

func TestIntegration_MultiStack(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--dir", dir, "--non-interactive"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	devcontainerDir := filepath.Join(dir, ".devcontainer")

	// All 8 files exist.
	for _, name := range expectedFiles {
		info := assertFileExists(t, filepath.Join(devcontainerDir, name))
		if info.Size() == 0 {
			t.Errorf("file is empty: %s", name)
		}
	}

	// Dockerfile: contains Go runtime, both LSP installs.
	dockerfile := readFile(t, filepath.Join(devcontainerDir, "Dockerfile"))
	if !strings.Contains(dockerfile, `go = "latest"`) {
		t.Error("Dockerfile should contain go runtime")
	}
	// Node is hardcoded (not in the range block), so we do NOT assert node = "lts" in range.
	if !strings.Contains(dockerfile, "go install golang.org/x/tools/gopls@latest") {
		t.Error("Dockerfile should contain gopls install")
	}
	if !strings.Contains(dockerfile, "npm install -g typescript-language-server typescript") {
		t.Error("Dockerfile should contain typescript-language-server install")
	}

	// init-firewall.sh: contains domains from both stacks.
	initFirewall := readFile(t, filepath.Join(devcontainerDir, "init-firewall.sh"))
	// registry.npmjs.org is Static in firewall registry, so it should appear in dig section.
	if !strings.Contains(initFirewall, "dig +short 'registry.npmjs.org'") {
		t.Error("init-firewall.sh should contain dig resolution for registry.npmjs.org")
	}

	// dynamic-domains.conf: contains both Go and Node dynamic domains.
	dynamicDomains := readFile(t, filepath.Join(devcontainerDir, "dynamic-domains.conf"))
	for _, domain := range []string{"proxy.golang.org", "cdn.jsdelivr.net"} {
		if !strings.Contains(dynamicDomains, domain) {
			t.Errorf("dynamic-domains.conf should contain %s", domain)
		}
	}

	// README.md: contains both stacks.
	readme := readFile(t, filepath.Join(devcontainerDir, "README.md"))
	if !strings.Contains(readme, "- go\n") {
		t.Error("README.md should list go")
	}
	if !strings.Contains(readme, "- node\n") {
		t.Error("README.md should list node")
	}

	// claude-user-settings.json: both plugins.
	// Unmarshal validates JSON syntax; values are checked via string assertions below.
	claudeSettings := readFile(t, filepath.Join(devcontainerDir, "claude-user-settings.json"))
	var claudeMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(claudeSettings), &claudeMap); err != nil {
		t.Fatalf("claude-user-settings.json is not valid JSON: %v", err)
	}
	if !strings.Contains(claudeSettings, `"gopls"`) {
		t.Error("claude-user-settings.json should contain gopls plugin")
	}
	if !strings.Contains(claudeSettings, `"typescript"`) {
		t.Error("claude-user-settings.json should contain typescript plugin")
	}
}

func TestIntegration_ExistingDevcontainerAborts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pre-create .devcontainer/ directory.
	devcontainerDir := filepath.Join(dir, ".devcontainer")
	if err := os.MkdirAll(devcontainerDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--dir", dir, "--non-interactive"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when .devcontainer/ already exists")
	}
	if !strings.Contains(err.Error(), ".devcontainer/ already exists") {
		t.Errorf("error should mention '.devcontainer/ already exists'; got: %s", err.Error())
	}

	// Directory should remain empty (no files written).
	entries, readErr := os.ReadDir(devcontainerDir)
	if readErr != nil {
		t.Fatalf("read .devcontainer/: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf(".devcontainer/ should be empty after abort; found %d entries", len(entries))
	}
}

func TestIntegration_ExtraDomains(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--dir", dir, "--non-interactive", "--extra-domains", "api.example.com"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	devcontainerDir := filepath.Join(dir, ".devcontainer")

	// All 8 files exist.
	for _, name := range expectedFiles {
		assertFileExists(t, filepath.Join(devcontainerDir, name))
	}

	// dynamic-domains.conf: contains the user-specified extra domain.
	dynamicDomains := readFile(t, filepath.Join(devcontainerDir, "dynamic-domains.conf"))
	if !strings.Contains(dynamicDomains, "api.example.com") {
		t.Error("dynamic-domains.conf should contain api.example.com")
	}

	// README.md: contains the extra domain.
	readme := readFile(t, filepath.Join(devcontainerDir, "README.md"))
	if !strings.Contains(readme, "api.example.com") {
		t.Error("README.md should contain api.example.com")
	}
}

func TestIntegration_ConfigFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Truncate to second precision because yaml.v3 truncates time.Time to
	// whole seconds during the write/load round-trip. Without truncation,
	// a nanosecond-precise startTime captured mid-second could be After
	// the truncated GeneratedAt value from the same second, causing flaky failures.
	startTime := time.Now().UTC().Truncate(time.Second)

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--dir", dir, "--non-interactive", "--extra-domains", "api.example.com"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// .agentbox.yml exists.
	cfgPath := filepath.Join(dir, ".agentbox.yml")
	assertFileExists(t, cfgPath)

	// Round-trip via config.Load.
	f, err := os.Open(cfgPath)
	if err != nil {
		t.Fatalf("open %s: %v", cfgPath, err)
	}
	defer f.Close()

	cfg, err := config.Load(f)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("version: got %d, want 1", cfg.Version)
	}

	if len(cfg.Stacks) != 1 || cfg.Stacks[0] != "go" {
		t.Errorf("stacks: got %v, want [go]", cfg.Stacks)
	}

	if len(cfg.ExtraDomains) != 1 || cfg.ExtraDomains[0] != "api.example.com" {
		t.Errorf("extra_domains: got %v, want [api.example.com]", cfg.ExtraDomains)
	}

	if cfg.AgentboxVersion != "dev" {
		t.Errorf("agentbox_version: got %q, want %q", cfg.AgentboxVersion, "dev")
	}

	// generated_at should be between startTime and now.
	if cfg.GeneratedAt.Before(startTime) || cfg.GeneratedAt.After(time.Now().UTC()) {
		t.Errorf("generated_at %v is not within expected range [%v, now]", cfg.GeneratedAt, startTime)
	}
}
