package render

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bjro/ccbox/internal/stack"
)

// claudeSettings mirrors the JSON structure of claude-user-settings.json
// for test unmarshaling.
type claudeSettings struct {
	Permissions struct {
		DefaultMode string   `json:"defaultMode"`
		Allow       []string `json:"allow"`
	} `json:"permissions"`
	EnabledPlugins []string `json:"enabledPlugins"`
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

	if settings.Permissions.DefaultMode != "bypassPermissions" {
		t.Errorf("defaultMode = %q, want %q", settings.Permissions.DefaultMode, "bypassPermissions")
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
	pluginSet := make(map[string]bool)
	for _, p := range settings.EnabledPlugins {
		pluginSet[p] = true
	}

	for _, lsp := range cfg.LSPs {
		if !pluginSet[lsp.Plugin] {
			t.Errorf("enabledPlugins missing plugin %q from LSP %q", lsp.Plugin, lsp.Package)
		}
	}

	// Reverse check: every enabledPlugin must come from the config.
	lspPlugins := make(map[string]bool)
	for _, lsp := range cfg.LSPs {
		lspPlugins[lsp.Plugin] = true
	}
	for _, p := range settings.EnabledPlugins {
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

	spotChecks := []string{"gopls", "typescript"}
	pluginSet := make(map[string]bool)
	for _, p := range settings.EnabledPlugins {
		pluginSet[p] = true
	}

	for _, expected := range spotChecks {
		if !pluginSet[expected] {
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

	seen := make(map[string]bool)
	for _, p := range settings.EnabledPlugins {
		if seen[p] {
			t.Errorf("duplicate plugin %q in enabledPlugins", p)
		}
		seen[p] = true
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

	// Check the raw JSON for the empty array form.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(files.UserSettings, &raw); err != nil {
		t.Fatalf("raw JSON parse: %v", err)
	}
	trimmed := strings.TrimSpace(string(raw["enabledPlugins"]))
	if trimmed != "[]" {
		t.Errorf("enabledPlugins is %q, want []", trimmed)
	}
}

func TestRenderClaude_DirectConfig_CustomPlugins(t *testing.T) {
	cfg := GenerationConfig{
		LSPs: []stack.LSP{
			{Package: "custom-lsp", Plugin: "custom-plugin"},
			{Package: "another-lsp", Plugin: "another-plugin"},
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

	pluginSet := make(map[string]bool)
	for _, p := range settings.EnabledPlugins {
		pluginSet[p] = true
	}

	if !pluginSet["custom-plugin"] {
		t.Error("enabledPlugins missing 'custom-plugin'")
	}
	if !pluginSet["another-plugin"] {
		t.Error("enabledPlugins missing 'another-plugin'")
	}
}

func TestRenderClaude_DirectConfig_PluginWithSpecialChars(t *testing.T) {
	// Verify the jsonString FuncMap helper properly escapes characters that
	// would produce invalid JSON if interpolated raw.
	cfg := GenerationConfig{
		LSPs: []stack.LSP{
			{Package: "tricky-lsp", Plugin: `quote"and\backslash`},
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
	if settings.EnabledPlugins[0] != want {
		t.Errorf("enabledPlugins[0] = %q, want %q", settings.EnabledPlugins[0], want)
	}
}
