package render

import (
	"strings"
	"testing"

	"github.com/bjro/agentbox/internal/firewall"
	"github.com/bjro/agentbox/internal/stack"
)

func TestREADME_NoError(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	if len(out) == 0 {
		t.Error("README output is empty")
	}
}

func TestREADME_ContainsAllSections(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	sections := []string{
		"## Overview",
		"## Prerequisites",
		"## Getting Started",
		"## Firewall Architecture",
		"## Adding Domains",
		"## Coding Tools",
		"## Coding Tool Permissions",
		"## Settings Sync",
		"## Customization",
		"## Updating",
		"## Troubleshooting",
	}
	for _, section := range sections {
		if !strings.Contains(out, section) {
			t.Errorf("README missing section heading %q", section)
		}
	}
}

func TestREADME_DetectedStacksListed(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	// Stack IDs should appear as Markdown list items to avoid false-positive
	// substring matches (e.g., "go" appearing in domain rationale text).
	if !strings.Contains(out, "- go\n") {
		t.Error("README missing stack 'go' as list item")
	}
	if !strings.Contains(out, "- node\n") {
		t.Error("README missing stack 'node' as list item")
	}
}

func TestREADME_StaticDomainsInTable(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	// Registry-computed completeness: every static domain must appear.
	for _, d := range cfg.Domains.Static {
		if !strings.Contains(out, d.Name) {
			t.Errorf("README missing static domain %q", d.Name)
		}
	}

	// Spot-check well-known static domain.
	if !strings.Contains(out, "github.com") {
		t.Error("README missing well-known static domain 'github.com'")
	}
}

func TestREADME_DynamicDomainsInTable(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	// Registry-computed completeness: every dynamic domain (stripped) must appear.
	for _, d := range cfg.Domains.Dynamic {
		stripped := strings.TrimPrefix(d.Name, "*.")
		if !strings.Contains(out, stripped) {
			t.Errorf("README missing dynamic domain %q (stripped from %q)", stripped, d.Name)
		}
	}
}

func TestREADME_EmptyConfig(t *testing.T) {
	cfg := GenerationConfig{
		Stacks:     []stack.StackID{},
		Runtimes:   []stack.Runtime{},
		LSPs:       []stack.LSP{},
		SystemDeps: []string{},
		Domains: firewall.MergedDomains{
			Static:  []firewall.Domain{},
			Dynamic: []firewall.Domain{},
		},
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	// Should produce valid Markdown without template artifacts.
	if strings.Contains(out, "<no value>") {
		t.Error("README contains '<no value>' template artifact")
	}
	if strings.Contains(out, "{{") {
		t.Error("README contains '{{' template artifact")
	}
	if strings.Contains(out, "}}") {
		t.Error("README contains '}}' template artifact")
	}
	if strings.Contains(out, "<nil>") {
		t.Error("README contains '<nil>' template artifact")
	}

	// Static sections should still be present.
	staticSections := []string{
		"## Overview",
		"## Prerequisites",
		"## Getting Started",
		"## Firewall Architecture",
		"## Adding Domains",
		"## Coding Tools",
		"## Coding Tool Permissions",
		"## Settings Sync",
		"## Customization",
		"## Updating",
		"## Troubleshooting",
	}
	for _, section := range staticSections {
		if !strings.Contains(out, section) {
			t.Errorf("empty config README missing section heading %q", section)
		}
	}

	// No domain tables should appear when domains are empty (template guards with {{ if }}).
	if strings.Contains(out, "| Domain") {
		t.Error("empty config README contains a domain table; expected tables to be skipped")
	}
}

func TestREADME_AllStacks(t *testing.T) {
	allIDs := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}
	cfg, err := Merge(allIDs, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	// All stack IDs should appear.
	for _, id := range allIDs {
		if !strings.Contains(out, string(id)) {
			t.Errorf("README missing stack %q", id)
		}
	}

	// Every static domain should appear.
	for _, d := range cfg.Domains.Static {
		if !strings.Contains(out, d.Name) {
			t.Errorf("README missing static domain %q", d.Name)
		}
	}

	// Every dynamic domain (stripped) should appear.
	for _, d := range cfg.Domains.Dynamic {
		stripped := strings.TrimPrefix(d.Name, "*.")
		if !strings.Contains(out, stripped) {
			t.Errorf("README missing dynamic domain %q (stripped from %q)", stripped, d.Name)
		}
	}
}

func TestREADME_NoTemplateArtifacts(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	artifacts := []string{"<no value>", "{{", "}}", "<nil>"}
	for _, a := range artifacts {
		if strings.Contains(out, a) {
			t.Errorf("README contains template artifact %q", a)
		}
	}
}

func TestREADME_NoDoNotEditFooter(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	if strings.Contains(out, "do not edit manually") {
		t.Error("README should not contain 'do not edit manually' footer")
	}
}

func TestREADME_UpdatingSection(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	if !strings.Contains(out, "agentbox update") {
		t.Error("README should document 'agentbox update' command")
	}
	if !strings.Contains(out, "mise-config.toml") {
		t.Error("README should mention mise-config.toml for version changes")
	}
	if !strings.Contains(out, "custom") {
		t.Error("README should document custom stage")
	}
}

func TestREADME_Deterministic(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out1, err := README(cfg)
	if err != nil {
		t.Fatalf("README (first): %v", err)
	}

	out2, err := README(cfg)
	if err != nil {
		t.Fatalf("README (second): %v", err)
	}

	if out1 != out2 {
		t.Error("README output is not deterministic; two renders differ")
	}
}

func TestREADME_WildcardDomainsStripped(t *testing.T) {
	// AlwaysOn includes *.anthropic.com as a Dynamic domain.
	cfg, err := Merge(nil, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	// The table should show "anthropic.com" (stripped), not "*.anthropic.com".
	if !strings.Contains(out, "anthropic.com") {
		t.Error("README missing stripped 'anthropic.com' for wildcard domain")
	}

	// Verify the raw wildcard form does not appear as a domain entry in a table row.
	// Table rows start with "| ", so check for "| *.anthropic.com".
	if strings.Contains(out, "| *.anthropic.com") {
		t.Error("README contains raw wildcard '*.anthropic.com' in table; should be stripped")
	}
}

func TestREADME_CodingToolsSection(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	checks := []string{
		"## Coding Tools",
		"Claude Code",
		"Codex CLI",
		"OPENAI_API_KEY",
		"ChatGPT",
		"codex",
		"codex --full-auto",
		"persist across container rebuilds",
	}
	for _, s := range checks {
		if !strings.Contains(out, s) {
			t.Errorf("Coding Tools section missing %q", s)
		}
	}
}

func TestREADME_CodingToolPermissionsSection(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	if !strings.Contains(out, "## Coding Tool Permissions") {
		t.Error("README missing '## Coding Tool Permissions' heading")
	}
	if !strings.Contains(out, "bypassPermissions") {
		t.Error("Permissions section missing 'bypassPermissions' for Claude Code")
	}
	if !strings.Contains(out, "danger-full-access") {
		t.Error("Permissions section missing 'danger-full-access' for Codex")
	}
}

func TestREADME_SettingsSyncMentionsBothTools(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	if !strings.Contains(out, "sync-claude-settings.sh") {
		t.Error("Settings Sync section missing 'sync-claude-settings.sh'")
	}
	if !strings.Contains(out, "sync-codex-settings.sh") {
		t.Error("Settings Sync section missing 'sync-codex-settings.sh'")
	}
}

func TestREADME_CustomizationContainerEnvForwarding(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	if !strings.Contains(out, "containerEnv") {
		t.Error("Customization section missing 'containerEnv'")
	}
	if !strings.Contains(out, "forwarded") {
		t.Error("Customization section missing forwarding semantics for containerEnv")
	}
}

func TestREADME_NilDomainSlices(t *testing.T) {
	// Hand-built config with zero-value MergedDomains (nil slices).
	cfg := GenerationConfig{
		Stacks:     []stack.StackID{},
		Runtimes:   []stack.Runtime{},
		LSPs:       []stack.LSP{},
		SystemDeps: []string{},
		Domains:    firewall.MergedDomains{},
	}

	out, err := README(cfg)
	if err != nil {
		t.Fatalf("README: %v", err)
	}

	if strings.Contains(out, "<no value>") {
		t.Error("README contains '<no value>' template artifact with nil domain slices")
	}
	if strings.Contains(out, "<nil>") {
		t.Error("README contains '<nil>' template artifact with nil domain slices")
	}

	// Static sections should still be present.
	if !strings.Contains(out, "## Overview") {
		t.Error("nil-domain README missing '## Overview' section")
	}
	if !strings.Contains(out, "## Firewall Architecture") {
		t.Error("nil-domain README missing '## Firewall Architecture' section")
	}
}
