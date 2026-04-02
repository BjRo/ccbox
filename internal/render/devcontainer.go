package render

import (
	"fmt"
	"io"
	"text/template"
)

// DevContainer renders the devcontainer.json configuration to w. The output
// is a static JSON document that wires together the Dockerfile, firewall
// scripts, and Claude settings produced by other render functions. The cfg
// parameter is accepted for API consistency with other render functions,
// even though the current template contains no dynamic actions.
func DevContainer(w io.Writer, cfg GenerationConfig) error {
	tmpl, err := template.ParseFS(templateFS, "templates/devcontainer.json.tmpl")
	if err != nil {
		return fmt.Errorf("render devcontainer.json: %w", err)
	}

	if err := tmpl.Execute(w, cfg); err != nil {
		return fmt.Errorf("render devcontainer.json: %w", err)
	}

	return nil
}
