package render

import (
	"strings"
	"testing"

	"github.com/bjro/agentbox/internal/firewall"
	"github.com/bjro/agentbox/internal/stack"
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
	if !strings.Contains(out, "FROM debian:bookworm-slim AS agentbox") {
		t.Error("output missing FROM debian:bookworm-slim AS agentbox")
	}
	if !strings.Contains(out, "AGENTBOX MANAGED -- DO NOT EDIT THIS STAGE") {
		t.Error("output missing managed stage header comment")
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

func TestDockerfile_MiseConfigCopied_SingleStack(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "COPY config.toml /home/node/.config/mise/config.toml") {
		t.Error("output missing COPY config.toml directive")
	}
}

func TestDockerfile_MiseConfigCopied_MultiStack(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "COPY config.toml /home/node/.config/mise/config.toml") {
		t.Error("output missing COPY config.toml directive")
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

	if !strings.Contains(out, "FROM debian:bookworm-slim AS agentbox") {
		t.Error("empty config missing base image with stage name")
	}
	if !strings.Contains(out, "build-essential") {
		t.Error("empty config missing system packages")
	}
	if !strings.Contains(out, "npm install -g @anthropic-ai/claude-code") {
		t.Error("empty config missing Claude Code install")
	}
	if !strings.Contains(out, "@openai/codex") {
		t.Error("empty config missing Codex CLI install")
	}
	if !strings.Contains(out, "COPY config.toml /home/node/.config/mise/config.toml") {
		t.Error("empty config missing COPY config.toml directive")
	}
	// No LSP installs should be present.
	if strings.Contains(out, "gopls") || strings.Contains(out, "typescript-language-server") {
		t.Error("empty config should not have LSP install commands")
	}
	// No dev tool installs should be present.
	if strings.Contains(out, "golangci-lint") {
		t.Error("empty config should not have dev tool install commands")
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

func TestDockerfile_CodexCLIInstall(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "@openai/codex") {
		t.Error("output missing Codex CLI install command")
	}
}

func TestDockerfile_CodexCLI_Ordering(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	claudeIdx := strings.Index(out, "@anthropic-ai/claude-code")
	codexIdx := strings.Index(out, "@openai/codex")
	userRootIdx := strings.Index(out, "USER root")

	if claudeIdx == -1 {
		t.Fatal("output missing Claude Code install")
	}
	if codexIdx == -1 {
		t.Fatal("output missing Codex CLI install")
	}
	if userRootIdx == -1 {
		t.Fatal("output missing USER root directive")
	}

	if claudeIdx > codexIdx {
		t.Error("Claude Code install should appear before Codex CLI install")
	}
	if codexIdx > userRootIdx {
		t.Error("Codex CLI install should appear before USER root directive")
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

	// File ends with exactly one trailing newline.
	if !strings.HasSuffix(out, "\n") {
		t.Error("output does not end with a trailing newline")
	}
	if strings.HasSuffix(out, "\n\n") {
		t.Error("output ends with double trailing newline")
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

func TestDockerfile_MiseConfigCopied(t *testing.T) {
	// COPY config.toml must appear for all stack combinations.
	// The node-always-present invariant is tested in ensure_test.go and mise_test.go.
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

			if !strings.Contains(out, "COPY config.toml /home/node/.config/mise/config.toml") {
				t.Error("output missing COPY config.toml directive")
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
		DevTools:   []string{},
		Domains:    firewall.MergedDomains{Static: []firewall.Domain{}, Dynamic: []firewall.Domain{}},
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "FROM debian:bookworm-slim AS agentbox") {
		t.Error("minimal config missing base image with stage name")
	}
	if !strings.Contains(out, "npm install -g @anthropic-ai/claude-code") {
		t.Error("minimal config missing Claude Code install")
	}
	if !strings.Contains(out, "@openai/codex") {
		t.Error("minimal config missing Codex CLI install")
	}
	if !strings.Contains(out, "COPY config.toml /home/node/.config/mise/config.toml") {
		t.Error("minimal config missing COPY config.toml directive")
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
			{Package: "zls", InstallCmd: "zig-install zls", Plugins: map[string]string{stack.CodingToolClaude: "zls"}},
		},
		SystemDeps: []string{"libfoo-dev"},
		DevTools:   []string{},
		Domains:    firewall.MergedDomains{Static: []firewall.Domain{}, Dynamic: []firewall.Domain{}},
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	// Runtime versions are no longer in Dockerfile; they live in config.toml.
	if !strings.Contains(out, "COPY config.toml /home/node/.config/mise/config.toml") {
		t.Error("output missing COPY config.toml directive")
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

	// COPY config.toml must be present (replaces inline mise config).
	if !strings.Contains(out, "COPY config.toml /home/node/.config/mise/config.toml") {
		t.Error("output missing COPY config.toml directive")
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

	// Structural assertion: every dev tool install command must appear.
	for _, dt := range cfg.DevTools {
		if !strings.Contains(out, dt) {
			t.Errorf("output missing dev tool install command %q", dt)
		}
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

func TestDockerfile_DevTools_GoStack(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if !strings.Contains(out, "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest") {
		t.Error("output missing golangci-lint install command for Go stack")
	}
}

func TestDockerfile_DevTools_AbsentForNonGoStack(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	if strings.Contains(out, "golangci-lint") {
		t.Error("Node-only output should not contain golangci-lint")
	}
	if strings.Contains(out, "Dev tools") {
		t.Error("Node-only output should not contain Dev tools section")
	}
}

func TestDockerfile_DevTools_OrderingInDockerfile(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := Dockerfile(cfg)
	if err != nil {
		t.Fatalf("Dockerfile: %v", err)
	}

	miseIdx := strings.Index(out, "RUN mise install")
	golangciIdx := strings.Index(out, "golangci-lint")
	claudeIdx := strings.Index(out, "npm install -g @anthropic-ai/claude-code")
	codexIdx := strings.Index(out, "@openai/codex")

	if miseIdx == -1 {
		t.Fatal("output missing mise install")
	}
	if golangciIdx == -1 {
		t.Fatal("output missing golangci-lint")
	}
	if claudeIdx == -1 {
		t.Fatal("output missing Claude Code install")
	}
	if codexIdx == -1 {
		t.Fatal("output missing Codex CLI install")
	}

	if golangciIdx < miseIdx {
		t.Error("golangci-lint should appear after mise install")
	}
	if golangciIdx > claudeIdx {
		t.Error("golangci-lint should appear before Claude Code install")
	}
	if golangciIdx > codexIdx {
		t.Error("golangci-lint should appear before Codex CLI install")
	}
}

func TestDockerfile_DevTools_NoTripleNewlines(t *testing.T) {
	// When DevTools is empty, the template should not produce triple newlines.
	for _, tc := range []struct {
		name   string
		stacks []stack.StackID
	}{
		{"empty stacks", []stack.StackID{}},
		{"node only", []stack.StackID{stack.Node}},
		{"go only", []stack.StackID{stack.Go}},
		{"all stacks", []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}},
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

			if strings.Contains(out, "\n\n\n") {
				t.Error("Dockerfile contains triple newlines")
			}
		})
	}
}

func TestDockerfile_DirectConfig_SystemDepsOnly(t *testing.T) {
	cfg := GenerationConfig{
		Stacks:     []stack.StackID{},
		Runtimes:   []stack.Runtime{},
		LSPs:       []stack.LSP{},
		SystemDeps: []string{"libbar-dev", "libquux-dev"},
		DevTools:   []string{},
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
