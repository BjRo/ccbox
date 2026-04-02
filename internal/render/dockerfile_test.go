package render

import (
	"strings"
	"testing"

	"github.com/bjro/ccbox/internal/firewall"
	"github.com/bjro/ccbox/internal/stack"
)

func TestDockerfile_BaseImage(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "FROM debian:bookworm-slim") {
		t.Error("output missing FROM debian:bookworm-slim")
	}
}

func TestDockerfile_AlwaysIncludedPackages(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	required := []string{"curl", "git", "sudo", "zsh", "iptables", "dnsmasq", "build-essential"}
	for _, pkg := range required {
		if !strings.Contains(out, pkg) {
			t.Errorf("output missing always-included package %q", pkg)
		}
	}
}

func TestDockerfile_MiseToolsSingleStack(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, `go = "latest"`) {
		t.Error(`output missing go = "latest" in mise config`)
	}
	// Node is always present (Claude Code dependency).
	if !strings.Contains(out, `node = "lts"`) {
		t.Error(`output missing node = "lts" in mise config (always included for Claude Code)`)
	}
	// Should not contain other stacks' tools.
	for _, absent := range []string{`python = "`, `ruby = "`, `rust = "`} {
		if strings.Contains(out, absent) {
			t.Errorf("output should not contain %q for Go-only config", absent)
		}
	}
}

func TestDockerfile_MiseToolsMultiStack(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	expected := []string{`go = "latest"`, `node = "lts"`, `python = "latest"`}
	for _, want := range expected {
		if !strings.Contains(out, want) {
			t.Errorf("output missing mise tool entry %q", want)
		}
	}
}

func TestDockerfile_LSPInstallCommands(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "go install golang.org/x/tools/gopls@latest") {
		t.Error("output missing gopls install command")
	}

	// Multi-stack: Go + Node
	cfg2, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out2, err := Dockerfile(cfg2)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out2, "npm install -g typescript-language-server typescript") {
		t.Error("output missing typescript-language-server install command")
	}
}

func TestDockerfile_SystemDepsIncluded(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Ruby}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "libssl-dev") {
		t.Error("output missing libssl-dev for Ruby stack")
	}
	if !strings.Contains(out, "libreadline-dev") {
		t.Error("output missing libreadline-dev for Ruby stack")
	}

	// Go-only should not have Ruby's system deps.
	goCfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	goOut, err := Dockerfile(goCfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if strings.Contains(goOut, "libssl-dev") {
		t.Error("Go-only output should not contain libssl-dev")
	}
}

func TestDockerfile_SystemDepsDeduplication(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Ruby, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	count := strings.Count(out, "libssl-dev")
	if count != 1 {
		t.Errorf("libssl-dev appears %d times, want exactly 1", count)
	}
}

func TestDockerfile_EmptyConfig(t *testing.T) {
	cfg, err := Merge([]stack.StackID{}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "FROM debian:bookworm-slim") {
		t.Error("empty config missing base image")
	}
	if !strings.Contains(out, "build-essential") {
		t.Error("empty config missing system packages")
	}
	if !strings.Contains(out, "npm install -g @anthropic-ai/claude-code") {
		t.Error("empty config missing Claude Code install")
	}
	// Node is always present (Claude Code dependency), even with no stacks.
	if !strings.Contains(out, `node = "lts"`) {
		t.Error(`empty config missing node = "lts" (always included for Claude Code)`)
	}
	// No other mise tools should be listed.
	if strings.Contains(out, `go = "`) || strings.Contains(out, `python = "`) {
		t.Error("empty config should not have non-Node mise tool entries")
	}
	// No LSP installs should be present.
	if strings.Contains(out, "gopls") || strings.Contains(out, "typescript-language-server") {
		t.Error("empty config should not have LSP install commands")
	}
}

func TestDockerfile_FirewallScriptsCopied(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	for _, file := range []string{"init-firewall.sh", "warmup-dns.sh", "dynamic-domains.conf"} {
		if !strings.Contains(out, "COPY "+file) {
			t.Errorf("output missing COPY %s", file)
		}
	}
}

func TestDockerfile_UserAndWorkdir(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	// The last two non-empty lines should be USER node and WORKDIR /workspace.
	var lastTwo []string
	for i := len(lines) - 1; i >= 0 && len(lastTwo) < 2; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			lastTwo = append([]string{line}, lastTwo...)
		}
	}

	if len(lastTwo) < 2 {
		t.Fatal("output has fewer than 2 non-empty lines at the end")
	}
	if lastTwo[0] != "USER node" {
		t.Errorf("second-to-last line = %q, want %q", lastTwo[0], "USER node")
	}
	if lastTwo[1] != "WORKDIR /workspace" {
		t.Errorf("last line = %q, want %q", lastTwo[1], "WORKDIR /workspace")
	}
}

func TestDockerfile_ClaudeCodeInstall(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "npm install -g @anthropic-ai/claude-code") {
		t.Error("output missing Claude Code install command")
	}
}

func TestDockerfile_NoTrailingWhitespace(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Ruby, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if line != strings.TrimRight(line, " \t") {
			t.Errorf("line %d has trailing whitespace: %q", i+1, line)
		}
	}
}

func TestDockerfile_AptGetValidShellSyntax(t *testing.T) {
	// Ensure the apt-get install block produces valid shell syntax.
	// The template must not produce:
	// - A bare backslash on its own line (dangling continuation)
	// - A blank line inside the RUN continuation block
	// - A line ending with double backslash (\\)
	for _, tc := range []struct {
		name   string
		stacks []stack.StackID
	}{
		{"no system deps", []stack.StackID{stack.Go}},
		{"with system deps", []stack.StackID{stack.Ruby}},
		{"multi stack system deps", []stack.StackID{stack.Ruby, stack.Python}},
		{"empty stacks", []stack.StackID{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := Merge(tc.stacks, nil)
			if err != nil {
				t.Fatalf("Merge: %v", err)
			}

			out, err := Dockerfile(cfg)
			if err != nil {
				t.Fatalf("Dockerfile: %v", err)
			}

			lines := strings.Split(out, "\n")
			inAptBlock := false
			for i, line := range lines {
				trimmed := strings.TrimSpace(line)

				if strings.Contains(line, "apt-get install -y --no-install-recommends") {
					inAptBlock = true
				}

				if inAptBlock {
					// A bare backslash on its own line means a dangling continuation.
					if trimmed == "\\" {
						t.Errorf("line %d: bare backslash (dangling continuation): %q", i+1, line)
					}
					// A line ending with double backslash is invalid.
					if strings.HasSuffix(trimmed, "\\\\") {
						t.Errorf("line %d: double backslash: %q", i+1, line)
					}
					// A blank line inside the RUN continuation breaks the command.
					if trimmed == "" {
						t.Errorf("line %d: blank line inside apt-get install block", i+1)
					}
				}

				if inAptBlock && strings.Contains(line, "rm -rf") {
					inAptBlock = false
				}
			}
		})
	}
}

// Tests that call Dockerfile() directly with hand-built GenerationConfig
// structs to isolate template rendering from merging logic.

func TestDockerfile_DirectConfig_MinimalValid(t *testing.T) {
	cfg := GenerationConfig{
		Stacks:     []stack.StackID{},
		Runtimes:   []stack.Runtime{},
		LSPs:       []stack.LSP{},
		SystemDeps: []string{},
		Domains:    firewall.MergedDomains{Static: []firewall.Domain{}, Dynamic: []firewall.Domain{}},
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "FROM debian:bookworm-slim") {
		t.Error("minimal config missing base image")
	}
	if !strings.Contains(out, "npm install -g @anthropic-ai/claude-code") {
		t.Error("minimal config missing Claude Code install")
	}
}

func TestDockerfile_DirectConfig_CustomRuntimesAndLSPs(t *testing.T) {
	cfg := GenerationConfig{
		Stacks: []stack.StackID{"custom"},
		Runtimes: []stack.Runtime{
			{Tool: "deno", Version: "1.40"},
			{Tool: "zig", Version: "0.12"},
		},
		LSPs: []stack.LSP{
			{Package: "zls", InstallCmd: "zig-install zls", Plugin: "zls"},
		},
		SystemDeps: []string{"libfoo-dev"},
		Domains:    firewall.MergedDomains{Static: []firewall.Domain{}, Dynamic: []firewall.Domain{}},
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, `deno = "1.40"`) {
		t.Error("output missing deno runtime in mise config")
	}
	if !strings.Contains(out, `zig = "0.12"`) {
		t.Error("output missing zig runtime in mise config")
	}
	if !strings.Contains(out, "zig-install zls") {
		t.Error("output missing zls install command")
	}
	if !strings.Contains(out, "libfoo-dev") {
		t.Error("output missing libfoo-dev system dep")
	}
}

func TestDockerfile_DirectConfig_SystemDepsOnly(t *testing.T) {
	cfg := GenerationConfig{
		Stacks:     []stack.StackID{},
		Runtimes:   []stack.Runtime{},
		LSPs:       []stack.LSP{},
		SystemDeps: []string{"libbar-dev", "libquux-dev"},
		Domains:    firewall.MergedDomains{Static: []firewall.Domain{}, Dynamic: []firewall.Domain{}},
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "libbar-dev") {
		t.Error("output missing libbar-dev")
	}
	if !strings.Contains(out, "libquux-dev") {
		t.Error("output missing libquux-dev")
	}

	// Verify no bare backslash line in the apt-get block.
	for i, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) == "\\" {
			t.Errorf("line %d: bare backslash (dangling continuation)", i+1)
		}
	}
}
