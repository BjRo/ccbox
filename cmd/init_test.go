package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCommand_GeneratesDevcontainer(t *testing.T) {
	// Create a temp dir with a go.mod to trigger Go stack detection.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--dir", dir})

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

func TestInitCommand_StackFlagRenamed(t *testing.T) {
	dir := t.TempDir()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--stack", "go,node", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify Dockerfile was generated.
	path := filepath.Join(dir, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(path); err != nil {
		t.Error("missing Dockerfile")
	}
}

func TestInitCommand_StackFlagWithSpaces(t *testing.T) {
	dir := t.TempDir()

	cmd := newRootCmd()
	// Cobra StringSlice splits on commas but preserves spaces.
	// The code should TrimSpace each value.
	cmd.SetArgs([]string{"init", "--stack", "go, node", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	path := filepath.Join(dir, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(path); err != nil {
		t.Error("missing Dockerfile")
	}
}

func TestInitCommand_ExtraDomainsFlagRenamed(t *testing.T) {
	dir := t.TempDir()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--stack", "go", "--extra-domains", "api.example.com", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify the extra domain appears in dynamic-domains.conf (user extras are dynamic domains).
	content, err := os.ReadFile(filepath.Join(dir, ".devcontainer", "dynamic-domains.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "api.example.com") {
		t.Error("extra domain api.example.com not found in dynamic-domains.conf")
	}
}

func TestInitCommand_DirFlag(t *testing.T) {
	// Create a temp dir with go.mod.
	targetDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(targetDir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run from a different directory, but point --dir to targetDir.
	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--dir", targetDir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify .devcontainer/ was created inside the target dir.
	path := filepath.Join(targetDir, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(path); err != nil {
		t.Error("missing Dockerfile in target dir")
	}
}

func TestInitCommand_DirFlag_NonExistent(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--dir", "/nonexistent/path/abc123"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-existent --dir")
	}
	if !strings.Contains(err.Error(), "/nonexistent/path/abc123") {
		t.Errorf("error should mention the path; got: %s", err.Error())
	}
}

func TestInitCommand_DirFlag_NotADirectory(t *testing.T) {
	// Create a temp file (not a directory).
	tmpFile := filepath.Join(t.TempDir(), "afile")
	if err := os.WriteFile(tmpFile, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--dir", tmpFile})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for --dir pointing to a file")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error should mention 'not a directory'; got: %s", err.Error())
	}
}

func TestInitCommand_DirFlag_DefaultsToWorkingDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Use t.Chdir (Go 1.24+) to temporarily change working directory.
	t.Chdir(dir)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Verify .devcontainer/ was created in the working directory.
	path := filepath.Join(dir, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(path); err != nil {
		t.Error("missing Dockerfile in working dir")
	}
}

func TestInitCommand_NonInteractiveFlag(t *testing.T) {
	dir := t.TempDir()

	t.Run("-y short flag", func(t *testing.T) {
		testDir := filepath.Join(dir, "short")
		if err := os.MkdirAll(testDir, 0o755); err != nil {
			t.Fatal(err)
		}

		cmd := newRootCmd()
		cmd.SetArgs([]string{"init", "--stack", "go", "-y", "--dir", testDir})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("init with -y: %v", err)
		}

		path := filepath.Join(testDir, ".devcontainer", "Dockerfile")
		if _, err := os.Stat(path); err != nil {
			t.Error("missing Dockerfile with -y flag")
		}
	})

	t.Run("--non-interactive long flag", func(t *testing.T) {
		testDir := filepath.Join(dir, "long")
		if err := os.MkdirAll(testDir, 0o755); err != nil {
			t.Fatal(err)
		}

		cmd := newRootCmd()
		cmd.SetArgs([]string{"init", "--stack", "go", "--non-interactive", "--dir", testDir})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("init with --non-interactive: %v", err)
		}

		path := filepath.Join(testDir, ".devcontainer", "Dockerfile")
		if _, err := os.Stat(path); err != nil {
			t.Error("missing Dockerfile with --non-interactive flag")
		}
	})
}

func TestInitCommand_InvalidStack(t *testing.T) {
	dir := t.TempDir()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--stack", "elixir", "--dir", dir})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid stack ID")
	}
	if !strings.Contains(err.Error(), "unknown stack") {
		t.Errorf("error should mention 'unknown stack'; got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "elixir") {
		t.Errorf("error should mention the invalid stack 'elixir'; got: %s", err.Error())
	}
	// Should list valid stacks.
	if !strings.Contains(err.Error(), "go") {
		t.Errorf("error should list valid stacks; got: %s", err.Error())
	}
}

func TestInitCommand_NoStacksDetected_ReturnsError(t *testing.T) {
	// Empty dir with no marker files.
	dir := t.TempDir()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--dir", dir})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no stacks detected")
	}
	if !strings.Contains(err.Error(), "no stacks detected") {
		t.Errorf("error should mention 'no stacks detected'; got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "--stack") {
		t.Errorf("error should reference --stack flag; got: %s", err.Error())
	}
}

func TestInitCommand_StackAndDirCombined(t *testing.T) {
	// Empty dir -- no marker files needed since --stack explicitly provides stacks.
	dir := t.TempDir()

	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", "--stack", "go", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	path := filepath.Join(dir, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(path); err != nil {
		t.Error("missing Dockerfile in target dir")
	}
}

func TestInitCommand_OldFlagNames_NotAccepted(t *testing.T) {
	dir := t.TempDir()

	t.Run("--stacks is rejected", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init", "--stacks", "go", "--dir", dir})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for old --stacks flag")
		}
		if !strings.Contains(err.Error(), "unknown flag") {
			t.Errorf("error should mention 'unknown flag'; got: %s", err.Error())
		}
	})

	t.Run("--domains is rejected", func(t *testing.T) {
		cmd := newRootCmd()
		cmd.SetArgs([]string{"init", "--domains", "example.com", "--dir", dir})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for old --domains flag")
		}
		if !strings.Contains(err.Error(), "unknown flag") {
			t.Errorf("error should mention 'unknown flag'; got: %s", err.Error())
		}
	})
}

func TestInitCommand_StackFlagEmptyValuesFiltered(t *testing.T) {
	dir := t.TempDir()

	cmd := newRootCmd()
	// "go,,node" should filter out the empty string in the middle.
	cmd.SetArgs([]string{"init", "--stack", "go,,node", "--dir", dir})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	path := filepath.Join(dir, ".devcontainer", "Dockerfile")
	if _, err := os.Stat(path); err != nil {
		t.Error("missing Dockerfile")
	}
}
