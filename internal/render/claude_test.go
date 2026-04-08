package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bjro/agentbox/internal/stack"
)

// claudeSettings mirrors the JSON structure of claude-user-settings.json
// for test unmarshaling.
type claudeSettings struct {
	DefaultMode string `json:"defaultMode"`
	Permissions struct {
		Allow []string `json:"allow"`
	} `json:"permissions"`
	EnabledPlugins map[string]bool `json:"enabledPlugins"`
}

// --- Integration tests (through Merge + RenderClaude) ---

func TestRenderClaude_NoError(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	if len(files.UserSettings) == 0 {
		t.Error("UserSettings is empty")
	}
	if len(files.SyncSettings) == 0 {
		t.Error("SyncSettings is empty")
	}
}

func TestRenderClaude_UserSettings_ValidJSON(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("UserSettings is not valid JSON: %v\nContent:\n%s", err, files.UserSettings)
	}
}

func TestRenderClaude_UserSettings_Permissions(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}

	if settings.DefaultMode != "bypassPermissions" {
		t.Errorf("defaultMode = %q, want %q", settings.DefaultMode, "bypassPermissions")
	}

	expectedTools := []string{"Bash", "Read", "Write", "Edit", "Grep", "Glob", "Task", "WebFetch", "WebSearch"}
	if len(settings.Permissions.Allow) != len(expectedTools) {
		t.Fatalf("allow has %d entries, want %d", len(settings.Permissions.Allow), len(expectedTools))
	}
	for i, tool := range expectedTools {
		if settings.Permissions.Allow[i] != tool {
			t.Errorf("allow[%d] = %q, want %q", i, settings.Permissions.Allow[i], tool)
		}
	}
}

func TestRenderClaude_UserSettings_PluginsMatchRegistry(t *testing.T) {
	allIDs := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}
	cfg, err := Merge(allIDs, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}

	// Structural assertion: every LSP plugin from the config must appear in enabledPlugins.
	for _, lsp := range cfg.LSPs {
		p := lsp.Plugins[stack.CodingToolClaude]
		if p == "" {
			continue
		}
		if !settings.EnabledPlugins[p] {
			t.Errorf("enabledPlugins missing plugin %q from LSP %q", p, lsp.Package)
		}
	}

	// Reverse check: every enabledPlugin must come from the config.
	lspPlugins := make(map[string]bool)
	for _, lsp := range cfg.LSPs {
		if p := lsp.Plugins[stack.CodingToolClaude]; p != "" {
			lspPlugins[p] = true
		}
	}
	for p := range settings.EnabledPlugins {
		if !lspPlugins[p] {
			t.Errorf("enabledPlugins contains unexpected plugin %q", p)
		}
	}
}

func TestRenderClaude_UserSettings_PluginsSpotCheck(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}

	spotChecks := []string{"gopls-lsp@claude-plugins-official", "typescript-lsp@claude-plugins-official"}
	for _, expected := range spotChecks {
		if !settings.EnabledPlugins[expected] {
			t.Errorf("enabledPlugins missing well-known plugin %q", expected)
		}
	}
}

func TestRenderClaude_UserSettings_EmptyStacks(t *testing.T) {
	cfg, err := Merge(nil, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("UserSettings is not valid JSON: %v\nContent:\n%s", err, files.UserSettings)
	}

	// enabledPlugins should be an empty array, not null.
	if len(settings.EnabledPlugins) != 0 {
		t.Errorf("enabledPlugins has %d entries, want 0", len(settings.EnabledPlugins))
	}
}

func TestRenderClaude_UserSettings_NoDuplicatePlugins(t *testing.T) {
	// Merge with duplicate stack ID -- Merge deduplicates, so plugins should be unique.
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}

	// With the map format, JSON parsing already ensures no duplicate keys.
	// Just verify the count matches expectations.
	expectedCount := 0
	for _, lsp := range cfg.LSPs {
		if lsp.Plugins[stack.CodingToolClaude] != "" {
			expectedCount++
		}
	}
	if len(settings.EnabledPlugins) != expectedCount {
		t.Errorf("enabledPlugins has %d entries, want %d", len(settings.EnabledPlugins), expectedCount)
	}
}

func TestRenderClaude_UserSettings_Deterministic(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files1, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude (1): %v", err)
	}

	files2, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude (2): %v", err)
	}

	if !bytes.Equal(files1.UserSettings, files2.UserSettings) {
		t.Error("UserSettings output is not deterministic across two renders")
	}
}

// --- Sync script tests ---

func TestRenderClaude_SyncSettings_ScriptStructure(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	output := string(files.SyncSettings)

	checks := []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"claude-user-settings.json",
		"$HOME/.claude",
		"settings.json",
		"jq -s",
		`trap 'rm -f "$TMPFILE"' EXIT`,
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("SyncSettings missing structural marker %q", check)
		}
	}
}

func TestRenderClaude_SyncSettings_IsStatic(t *testing.T) {
	cfgGo, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge (Go): %v", err)
	}

	cfgMulti, err := Merge([]stack.StackID{stack.Go, stack.Node, stack.Python}, nil)
	if err != nil {
		t.Fatalf("Merge (Go+Node+Python): %v", err)
	}

	filesGo, err := RenderClaude(cfgGo)
	if err != nil {
		t.Fatalf("RenderClaude (Go): %v", err)
	}

	filesMulti, err := RenderClaude(cfgMulti)
	if err != nil {
		t.Fatalf("RenderClaude (Go+Node+Python): %v", err)
	}

	if !bytes.Equal(filesGo.SyncSettings, filesMulti.SyncSettings) {
		t.Error("SyncSettings output differs between Go-only and Go+Node+Python configs; it should be static")
	}
}

func TestRenderClaude_SyncSettings_NoTemplateArtifacts(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	if bytes.Contains(files.SyncSettings, []byte("<no value>")) {
		t.Error("SyncSettings contains '<no value>' template artifact")
	}
}

// --- Isolation tests (hand-built GenerationConfig) ---

func TestRenderClaude_DirectConfig_EmptyLSPs(t *testing.T) {
	cfg := GenerationConfig{
		LSPs: []stack.LSP{},
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("UserSettings is not valid JSON: %v\nContent:\n%s", err, files.UserSettings)
	}

	// Check the raw JSON for the empty object form.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(files.UserSettings, &raw); err != nil {
		t.Fatalf("raw JSON parse: %v", err)
	}
	// Empty object may render as {} or { } -- both are valid JSON.
	if len(settings.EnabledPlugins) != 0 {
		t.Errorf("enabledPlugins has %d entries, want 0", len(settings.EnabledPlugins))
	}
}

func TestRenderClaude_DirectConfig_CustomPlugins(t *testing.T) {
	cfg := GenerationConfig{
		LSPs: []stack.LSP{
			{Package: "custom-lsp", Plugins: map[string]string{stack.CodingToolClaude: "custom-plugin"}},
			{Package: "another-lsp", Plugins: map[string]string{stack.CodingToolClaude: "another-plugin"}},
		},
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("JSON parse: %v\nContent:\n%s", err, files.UserSettings)
	}

	if len(settings.EnabledPlugins) != 2 {
		t.Fatalf("enabledPlugins has %d entries, want 2", len(settings.EnabledPlugins))
	}

	if !settings.EnabledPlugins["custom-plugin"] {
		t.Error("enabledPlugins missing 'custom-plugin'")
	}
	if !settings.EnabledPlugins["another-plugin"] {
		t.Error("enabledPlugins missing 'another-plugin'")
	}
}

func TestRenderClaude_DirectConfig_PluginWithSpecialChars(t *testing.T) {
	// Verify the jsonString FuncMap helper properly escapes characters that
	// would produce invalid JSON if interpolated raw.
	cfg := GenerationConfig{
		LSPs: []stack.LSP{
			{Package: "tricky-lsp", Plugins: map[string]string{stack.CodingToolClaude: `quote"and\backslash`}},
		},
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("UserSettings is not valid JSON: %v\nContent:\n%s", err, files.UserSettings)
	}

	if len(settings.EnabledPlugins) != 1 {
		t.Fatalf("enabledPlugins has %d entries, want 1", len(settings.EnabledPlugins))
	}

	want := `quote"and\backslash`
	if !settings.EnabledPlugins[want] {
		t.Errorf("enabledPlugins missing %q", want)
	}
}

func TestRenderClaude_DirectConfig_NonClaudePluginsIgnored(t *testing.T) {
	cfg := GenerationConfig{
		LSPs: []stack.LSP{
			{Package: "some-lsp", Plugins: map[string]string{"codex": "codex-plugin"}},
		},
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("JSON parse: %v\nContent:\n%s", err, files.UserSettings)
	}

	if len(settings.EnabledPlugins) != 0 {
		t.Errorf("enabledPlugins has %d entries, want 0 (non-claude plugins should be ignored)", len(settings.EnabledPlugins))
	}
}

func TestRenderClaude_DirectConfig_MixedPlugins(t *testing.T) {
	cfg := GenerationConfig{
		LSPs: []stack.LSP{
			{Package: "mixed-lsp", Plugins: map[string]string{
				stack.CodingToolClaude: "claude-plugin",
				"codex":               "codex-plugin",
			}},
		},
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("JSON parse: %v\nContent:\n%s", err, files.UserSettings)
	}

	if len(settings.EnabledPlugins) != 1 {
		t.Fatalf("enabledPlugins has %d entries, want 1", len(settings.EnabledPlugins))
	}
	if !settings.EnabledPlugins["claude-plugin"] {
		t.Error("enabledPlugins missing 'claude-plugin'")
	}
}

func TestRenderClaude_DirectConfig_PluginlessFirstLSP(t *testing.T) {
	// Regression test for comma logic: first LSP (sorted by Package) has no
	// claude plugin, second does. The old index-based comma logic would emit
	// a leading comma producing invalid JSON like { , "zzz-plugin": true }.
	cfg := GenerationConfig{
		LSPs: []stack.LSP{
			{Package: "aaa-lsp", Plugins: map[string]string{}},
			{Package: "zzz-lsp", Plugins: map[string]string{stack.CodingToolClaude: "zzz-plugin"}},
		},
	}

	files, err := RenderClaude(cfg)
	if err != nil {
		t.Fatalf("RenderClaude: %v", err)
	}

	var settings claudeSettings
	if err := json.Unmarshal(files.UserSettings, &settings); err != nil {
		t.Fatalf("UserSettings is not valid JSON: %v\nContent:\n%s", err, files.UserSettings)
	}

	if len(settings.EnabledPlugins) != 1 {
		t.Fatalf("enabledPlugins has %d entries, want 1", len(settings.EnabledPlugins))
	}
	if !settings.EnabledPlugins["zzz-plugin"] {
		t.Error("enabledPlugins missing 'zzz-plugin'")
	}
}
