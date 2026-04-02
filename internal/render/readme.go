package render

import (
	"bytes"
	"fmt"
	"text/template"
)

var readmeTmpl = template.Must(
	template.New("").Funcs(funcMap).ParseFS(templateFS, "templates/README.md.tmpl"),
)

// README renders the README.md template using the given GenerationConfig.
// It returns the rendered content as a string. The README documents the
// generated devcontainer setup, including detected stacks, domain allowlists,
// firewall architecture, and troubleshooting guidance.
func README(cfg GenerationConfig) (string, error) {
	var buf bytes.Buffer
	if err := readmeTmpl.ExecuteTemplate(&buf, "README.md.tmpl", cfg); err != nil {
		return "", fmt.Errorf("render: execute README.md template: %w", err)
	}
	return buf.String(), nil
}
