package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bjro/ccbox/internal/stack"
	"github.com/bjro/ccbox/internal/wizard"
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

	// Pass nil prompter: non-TTY test stdin causes auto-detect fallback.
	cmd := newRootCmd(nil)
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

	cmd := newRootCmd(nil)
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
			Stacks: []stack.StackID{stack.Go},
		},
		failIfCalled: true,
	}

	cmd := newRootCmd(fake)
	cmd.SetArgs([]string{"init", "--stacks", "go"})

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
			Stacks:       []stack.StackID{stack.Go},
			ExtraDomains: []string{"api.example.com"},
		},
	}

	cmd := newRootCmd(fake)
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify the .devcontainer/ was generated.
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

	// Should succeed (empty stacks prints a message and returns nil).
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify no .devcontainer/ was created.
	devcontainerDir := filepath.Join(dir, ".devcontainer")
	if _, err := os.Stat(devcontainerDir); err == nil {
		t.Error(".devcontainer/ should not exist when wizard returns empty stacks")
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
