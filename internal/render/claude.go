package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"
)

// claudeFuncMap provides template helper functions for Claude settings rendering.
var claudeFuncMap = template.FuncMap{
	// jsonString JSON-encodes a string and strips the surrounding quotes,
	// producing a value safe for interpolation inside JSON double quotes.
	// This prevents invalid JSON from plugin names containing ", \, or
	// control characters.
	"jsonString": func(s string) (string, error) {
		b, err := json.Marshal(s)
		if err != nil {
			return "", err
		}
		// json.Marshal wraps the string in quotes; strip them.
		return string(b[1 : len(b)-1]), nil
	},
}

// claudeTemplates holds the parsed templates for Claude Code settings files.
// Parsing at package level via template.Must ensures syntax errors are caught
// at program startup. FuncMap must be registered before parsing so the parser
// recognizes custom functions.
var claudeTemplates = template.Must(
	template.New("").Funcs(claudeFuncMap).ParseFS(templateFS, "templates/claude-user-settings.json.tmpl", "templates/sync-claude-settings.sh.tmpl"),
)

// ClaudeFiles holds the rendered output of the two Claude-related files.
// Each field contains the full file content ready for writing to disk.
type ClaudeFiles struct {
	UserSettings []byte // claude-user-settings.json content
	SyncSettings []byte // sync-claude-settings.sh content
}

// RenderClaude executes the Claude templates against the given
// GenerationConfig and returns the rendered file contents. It is a pure
// transformation from config to bytes with no file I/O.
func RenderClaude(cfg GenerationConfig) (ClaudeFiles, error) {
	var userBuf, syncBuf bytes.Buffer

	if err := claudeTemplates.ExecuteTemplate(&userBuf, "claude-user-settings.json.tmpl", cfg); err != nil {
		return ClaudeFiles{}, fmt.Errorf("render claude-user-settings.json: %w", err)
	}

	if err := claudeTemplates.ExecuteTemplate(&syncBuf, "sync-claude-settings.sh.tmpl", cfg); err != nil {
		return ClaudeFiles{}, fmt.Errorf("render sync-claude-settings.sh: %w", err)
	}

	return ClaudeFiles{
		UserSettings: userBuf.Bytes(),
		SyncSettings: syncBuf.Bytes(),
	}, nil
}
