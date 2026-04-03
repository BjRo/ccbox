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

	if !strings.Contains(out, "# syntax=docker/dockerfile:1") {
		t.Error("output missing BuildKit syntax directive")
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

	for _, file := range []string{"init-firewall.sh", "warmup-dns.sh"} {
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

	// USER node should appear and WORKDIR /workspace should be the last non-empty line.
	if !strings.Contains(out, "USER node") {
		t.Error("output missing USER node")
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	var lastLine string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			lastLine = line
			break
		}
	}
	if lastLine != "WORKDIR /workspace" {
		t.Errorf("last non-empty line = %q, want %q", lastLine, "WORKDIR /workspace")
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

func TestDockerfile_GitDeltaUsesArchDetection(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	// git-delta must use dpkg --print-architecture, not hardcoded amd64.
	if strings.Contains(out, "git-delta_0.18.2_amd64.deb") {
		t.Error("git-delta URL still hardcodes amd64; should use dpkg --print-architecture")
	}
	if !strings.Contains(out, "dpkg --print-architecture") {
		t.Error("git-delta section missing dpkg --print-architecture")
	}
	if !strings.Contains(out, "git-delta_0.18.2_${ARCH}.deb") {
		t.Error("git-delta URL missing ${ARCH} variable substitution")
	}
}

func TestDockerfile_MiseInstallAsNodeUser(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	// mise install must run after USER node, not as root.
	nodeIdx := strings.Index(out, "USER node\n")
	miseInstallIdx := strings.Index(out, "RUN mise install")
	if nodeIdx == -1 {
		t.Fatal("output missing USER node directive before mise install")
	}
	if miseInstallIdx == -1 {
		t.Fatal("output missing RUN mise install")
	}
	if miseInstallIdx < nodeIdx {
		t.Error("mise install runs before USER node; should run as node user")
	}

	// mise binary should be copied to /usr/local/bin for all-user access.
	if !strings.Contains(out, "cp /root/.local/bin/mise /usr/local/bin/mise") {
		t.Error("output missing mise binary copy to /usr/local/bin")
	}
}

func TestDockerfile_NodeAlwaysInMiseConfig(t *testing.T) {
	// Even with no Node stack, node = "lts" must appear in mise config.
	for _, tc := range []struct {
		name   string
		stacks []stack.StackID
	}{
		{"go only", []stack.StackID{stack.Go}},
		{"rust only", []stack.StackID{stack.Rust}},
		{"empty stacks", []stack.StackID{}},
		{"node included", []stack.StackID{stack.Node}},
		{"go and node", []stack.StackID{stack.Go, stack.Node}},
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

			if !strings.Contains(out, `node = "lts"`) {
				t.Errorf("output missing node = \"lts\" in mise config")
			}

			// node = "lts" should appear exactly once (not duplicated).
			count := strings.Count(out, `node = "lts"`)
			if count != 1 {
				t.Errorf(`node = "lts" appears %d times, want exactly 1`, count)
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

func TestDockerfile_AllStacks(t *testing.T) {
	allIDs := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}
	cfg, err := Merge(allIDs, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	// Structural assertion: every runtime from the config must appear in mise config.
	// Node is handled specially (always hardcoded as node = "lts"), so we check
	// non-node runtimes via their tool = "version" format.
	for _, rt := range cfg.Runtimes {
		if rt.Tool == "node" {
			continue
		}
		expected := rt.Tool + ` = "` + rt.Version + `"`
		if !strings.Contains(out, expected) {
			t.Errorf("output missing mise runtime entry %q", expected)
		}
	}

	// Structural assertion: every LSP install command must appear.
	for _, lsp := range cfg.LSPs {
		if !strings.Contains(out, lsp.InstallCmd) {
			t.Errorf("output missing LSP install command %q", lsp.InstallCmd)
		}
	}

	// Structural assertion: every system dep must appear.
	for _, dep := range cfg.SystemDeps {
		if !strings.Contains(out, dep) {
			t.Errorf("output missing system dep %q", dep)
		}
	}

	// Node must appear exactly once in the mise config (not duplicated by
	// explicit Node stack inclusion).
	count := strings.Count(out, `node = "lts"`)
	if count != 1 {
		t.Errorf(`node = "lts" appears %d times, want exactly 1`, count)
	}

	// Spot-checks for well-known entries.
	spotChecks := []string{
		"go install golang.org/x/tools/gopls@latest",
		"pip install pyright",
		"gem install solargraph",
		"rustup component add rust-analyzer",
		"npm install -g typescript-language-server typescript",
	}
	for _, check := range spotChecks {
		if !strings.Contains(out, check) {
			t.Errorf("output missing well-known LSP install %q", check)
		}
	}
}

func TestDockerfile_NoTemplateArtifacts(t *testing.T) {
	allIDs := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}
	cfg, err := Merge(allIDs, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	artifacts := []string{"<no value>", "<nil>", "{{", "}}"}
	for _, a := range artifacts {
		if strings.Contains(out, a) {
			t.Errorf("Dockerfile contains template artifact %q", a)
		}
	}
}

func TestDockerfile_Deterministic(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out1, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile (first): %v", err)
	}

	out2, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile (second): %v", err)
	}

	if out1 != out2 {
		t.Error("Dockerfile output is not deterministic; two renders differ")
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
