package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bjro/agentbox/internal/stack"
	"github.com/bjro/agentbox/internal/wizard"
)

// expectedFiles lists the 9 files that agentbox init generates inside .devcontainer/.
// Intentionally coupled with the file map in cmd/init.go's RunE -- update both together.
var expectedInitFiles = []string{
	"Dockerfile",
	"devcontainer.json",
	"init-firewall.sh",
	"warmup-dns.sh",
	"dynamic-domains.conf",
	"claude-user-settings.json",
	"sync-claude-settings.sh",
	"README.md",
	"config.toml",
}

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

	// Pass nil prompter: non-TTY test stdin causes auto-detect fallback.
	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify key files were generated.
	devcontainerDir := filepath.Join(dir, ".devcontainer")
	for _, name := range expectedInitFiles {
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

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--stack", "go,node"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify Dockerfile was generated.
	path := filepath.Join(dir, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(path); err != nil {
		t.Error("missing Dockerfile")
	}

	// Verify config.toml was generated.
	configPath := filepath.Join(dir, ".devcontainer", "config.toml")
	if _, err := os.Stat(configPath); err != nil {
		t.Error("missing config.toml")
	}
}

func TestInitCommand_StacksFlagSkipsWizard(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Use a fake prompter that fatals if called.
	fake := &fakePrompter{
		t:   t,
		err: nil,
		choices: wizard.Choices{
			Stacks:          []stack.StackID{stack.Go},
			RuntimeVersions: nil,
		},
		failIfCalled: true,
	}

	cmd := newRootCmd(fake)
	cmd.SetArgs([]string{"init", "--stack", "go"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify output was generated (wizard was skipped, --stacks used directly).
	path := filepath.Join(dir, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(path); err != nil {
		t.Error("missing Dockerfile")
	}
}

func TestInitCommand_NonTTYSkipsWizard(t *testing.T) {
	dir := t.TempDir()

	// Create a go.mod so auto-detect finds something.
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Pass nil prompter: the non-TTY test stdin means the wizard
	// will not be instantiated, falling through to auto-detect.
	cmd := newRootCmd(nil)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify output was generated using auto-detect fallback.
	path := filepath.Join(dir, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(path); err != nil {
		t.Error("missing Dockerfile")
	}
}

func TestInitCommand_WizardFlow(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Fake prompter returns canned choices.
	fake := &fakePrompter{
		choices: wizard.Choices{
			Stacks:          []stack.StackID{stack.Go},
			ExtraDomains:    []string{"api.example.com"},
			RuntimeVersions: nil,
		},
	}

	cmd := newRootCmd(fake)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify the .devcontainer/ was generated.
	devcontainerDir := filepath.Join(dir, ".devcontainer")
	for _, name := range expectedInitFiles {
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

func TestInitCommand_WizardAborted(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Fake prompter returns ErrAborted.
	fake := &fakePrompter{
		err: wizard.ErrAborted,
	}

	cmd := newRootCmd(fake)
	cmd.SetArgs([]string{"init"})

	// Execute should succeed (abort is not an error).
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify no .devcontainer/ was created.
	devcontainerDir := filepath.Join(dir, ".devcontainer")
	if _, err := os.Stat(devcontainerDir); err == nil {
		t.Error(".devcontainer/ should not exist after wizard abort")
	}
}

func TestInitCommand_WizardEmptyStacks(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Fake prompter returns empty stacks (per finding 2, guard should catch this).
	fake := &fakePrompter{
		choices: wizard.Choices{
			Stacks:       []stack.StackID{},
			ExtraDomains: nil,
		},
	}

	cmd := newRootCmd(fake)
	cmd.SetArgs([]string{"init"})

	// Should return an error since no stacks were selected.
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for empty stacks")
	}
	if !strings.Contains(err.Error(), "no stacks") {
		t.Errorf("error should mention 'no stacks'; got: %s", err.Error())
	}

	// Verify no .devcontainer/ was created.
	devcontainerDir := filepath.Join(dir, ".devcontainer")
	if _, err := os.Stat(devcontainerDir); err == nil {
		t.Error(".devcontainer/ should not exist when wizard returns empty stacks")
	}
}

func TestInitCommand_ConfigTomlExists(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--stack", "go"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	configPath := filepath.Join(dir, ".devcontainer", "config.toml")
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal("config.toml not generated")
	}
	if info.Size() == 0 {
		t.Error("config.toml is empty")
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	if !strings.Contains(string(content), `go = "latest"`) {
		t.Error(`config.toml missing go = "latest"`)
	}
	if !strings.Contains(string(content), `node = "lts"`) {
		t.Error(`config.toml missing node = "lts"`)
	}
}

func TestInitCommand_RuntimeVersionFlag(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd(nil)
	cmd.SetArgs([]string{"init", "--stack", "go", "--runtime-version", "go=1.22,node=20"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	configPath := filepath.Join(dir, ".devcontainer", "config.toml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	if !strings.Contains(string(content), `go = "1.22"`) {
		t.Error(`config.toml missing go = "1.22"`)
	}
	if !strings.Contains(string(content), `node = "20"`) {
		t.Error(`config.toml missing node = "20"`)
	}
}

func TestInitCommand_RuntimeVersionFlagInvalid(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
	}{
		{"missing equals", "golatest"},
		{"empty tool", "=1.22"},
		{"empty version", "go="},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()

			orig, _ := os.Getwd()
			t.Cleanup(func() { _ = os.Chdir(orig) })
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}

			cmd := newRootCmd(nil)
			cmd.SetArgs([]string{"init", "--stack", "go", "--runtime-version", tc.value})

			err := cmd.Execute()
			if err == nil {
				t.Fatal("expected error for invalid --runtime-version")
			}
			if !strings.Contains(err.Error(), "invalid --runtime-version") {
				t.Errorf("error should mention 'invalid --runtime-version'; got: %s", err.Error())
			}
		})
	}
}

func TestInitCommand_WizardVersionOverrides(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Fake prompter returns custom runtime versions.
	fake := &fakePrompter{
		choices: wizard.Choices{
			Stacks:          []stack.StackID{stack.Go},
			RuntimeVersions: map[string]string{"go": "1.21"},
		},
	}

	cmd := newRootCmd(fake)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	configPath := filepath.Join(dir, ".devcontainer", "config.toml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	if !strings.Contains(string(content), `go = "1.21"`) {
		t.Error(`config.toml missing go = "1.21"`)
	}
}

func TestInitCommand_WizardAndFlagMerge(t *testing.T) {
	dir := t.TempDir()

	orig, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(orig) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Fake prompter returns custom runtime versions.
	fake := &fakePrompter{
		choices: wizard.Choices{
			Stacks:          []stack.StackID{stack.Go},
			RuntimeVersions: map[string]string{"go": "1.21", "node": "18"},
		},
	}

	cmd := newRootCmd(fake)
	// CLI flag overrides node but not go.
	cmd.SetArgs([]string{"init", "--runtime-version", "node=20"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	configPath := filepath.Join(dir, ".devcontainer", "config.toml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config.toml: %v", err)
	}
	// go should be from wizard.
	if !strings.Contains(string(content), `go = "1.21"`) {
		t.Error(`config.toml missing go = "1.21" (from wizard)`)
	}
	// node should be from CLI flag (overrides wizard).
	if !strings.Contains(string(content), `node = "20"`) {
		t.Error(`config.toml missing node = "20" (CLI flag should override wizard)`)
	}
}

func TestParseRuntimeVersions(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		input   []string
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "valid single pair",
			input: []string{"go=1.22"},
			want:  map[string]string{"go": "1.22"},
		},
		{
			name:  "valid multiple pairs",
			input: []string{"go=1.22", "node=20"},
			want:  map[string]string{"go": "1.22", "node": "20"},
		},
		{
			name:  "whitespace trimmed",
			input: []string{" go = 1.22 "},
			want:  map[string]string{"go": "1.22"},
		},
		{
			name:    "missing equals",
			input:   []string{"golatest"},
			wantErr: true,
		},
		{
			name:    "empty tool",
			input:   []string{"=1.22"},
			wantErr: true,
		},
		{
			name:    "empty version",
			input:   []string{"go="},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseRuntimeVersions(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d entries, want %d", len(got), len(tc.want))
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("got[%q] = %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestRenderFiles_ReturnsAllExpectedKeys(t *testing.T) {
	t.Parallel()

	files, err := renderFiles([]stack.StackID{stack.Go}, nil, nil)
	if err != nil {
		t.Fatalf("renderFiles: %v", err)
	}

	// Intentionally coupled with expectedInitFiles -- update both together.
	for _, name := range expectedInitFiles {
		if _, ok := files[name]; !ok {
			t.Errorf("missing key %q in renderFiles output", name)
		}
	}
}

func TestRenderFiles_DockerfileContainsASAgentbox(t *testing.T) {
	t.Parallel()

	files, err := renderFiles([]stack.StackID{stack.Go}, nil, nil)
	if err != nil {
		t.Fatalf("renderFiles: %v", err)
	}

	df := string(files["Dockerfile"])
	if !strings.Contains(df, "AS agentbox") {
		t.Error("Dockerfile should contain AS agentbox")
	}
	if strings.Contains(df, "FROM agentbox AS custom") {
		t.Error("Dockerfile should NOT contain custom stage (caller's responsibility)")
	}
}

func TestRenderFiles_AllValuesNonEmpty(t *testing.T) {
	t.Parallel()

	files, err := renderFiles([]stack.StackID{stack.Go}, nil, nil)
	if err != nil {
		t.Fatalf("renderFiles: %v", err)
	}

	for name, content := range files {
		if len(content) == 0 {
			t.Errorf("file %q has empty content", name)
		}
	}
}

func TestRenderFiles_VersionOverridesApplied(t *testing.T) {
	t.Parallel()

	overrides := map[string]string{"go": "1.22"}
	files, err := renderFiles([]stack.StackID{stack.Go}, nil, overrides)
	if err != nil {
		t.Fatalf("renderFiles: %v", err)
	}

	configToml := string(files["config.toml"])
	if !strings.Contains(configToml, `go = "1.22"`) {
		t.Error(`config.toml should contain go = "1.22"`)
	}
}

func TestRenderFiles_NilVersionOverrides(t *testing.T) {
	t.Parallel()

	files, err := renderFiles([]stack.StackID{stack.Go}, nil, nil)
	if err != nil {
		t.Fatalf("renderFiles: %v", err)
	}

	configToml := string(files["config.toml"])
	if !strings.Contains(configToml, `go = "latest"`) {
		t.Error(`config.toml should use default go = "latest" with nil overrides`)
	}
}

func TestRenderFiles_InvalidStack(t *testing.T) {
	t.Parallel()

	_, err := renderFiles([]stack.StackID{"nonexistent"}, nil, nil)
	if err == nil {
		t.Error("expected error for invalid stack ID")
	}
}

// fakePrompter is a test double for wizard.Prompter.
type fakePrompter struct {
	t            *testing.T
	choices      wizard.Choices
	err          error
	failIfCalled bool
}

func (f *fakePrompter) Run(_ []stack.StackID) (wizard.Choices, error) {
	if f.failIfCalled {
		f.t.Fatal("wizard should not have been called")
	}
	return f.choices, f.err
}
