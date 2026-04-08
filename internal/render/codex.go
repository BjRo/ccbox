package render

import (
	"bytes"
	"fmt"
	"text/template"
)

// codexTemplates holds the parsed templates for Codex CLI settings files.
// Both templates are static (no Go template actions), so no FuncMap is needed.
// Parsing at package level via template.Must ensures syntax errors are caught
// at program startup.
var codexTemplates = template.Must(
	template.ParseFS(templateFS, "templates/codex-config.toml.tmpl", "templates/sync-codex-settings.sh.tmpl"),
)

// CodexFiles holds the rendered output of the two Codex-related files.
// Each field contains the full file content ready for writing to disk.
type CodexFiles struct {
	Config       []byte // codex-config.toml content
	SyncSettings []byte // sync-codex-settings.sh content
}

// RenderCodex executes the Codex templates against the given
// GenerationConfig and returns the rendered file contents. It is a pure
// transformation from config to bytes with no file I/O.
//
// NOTE: The generated sync-codex-settings.sh requires devcontainer.json
// wiring (postStartCommand invocation and a ~/.codex volume mount) to take
// effect at container start. That integration is tracked in bean agentbox-0w8k.
func RenderCodex(cfg GenerationConfig) (CodexFiles, error) {
	var configBuf, syncBuf bytes.Buffer

	if err := codexTemplates.ExecuteTemplate(&configBuf, "codex-config.toml.tmpl", cfg); err != nil {
		return CodexFiles{}, fmt.Errorf("render codex-config.toml: %w", err)
	}

	if err := codexTemplates.ExecuteTemplate(&syncBuf, "sync-codex-settings.sh.tmpl", cfg); err != nil {
		return CodexFiles{}, fmt.Errorf("render sync-codex-settings.sh: %w", err)
	}

	return CodexFiles{
		Config:       configBuf.Bytes(),
		SyncSettings: syncBuf.Bytes(),
	}, nil
}
