package render

import (
	"fmt"
	"io"
	"text/template"
)

// devcontainerTmpl is parsed once at package init from the embedded template.
// The template is immutable (baked into the binary at compile time), so
// template.Must is the correct idiom: a parse failure is always a programmer
// error and should surface immediately at startup.
var devcontainerTmpl = template.Must(template.ParseFS(templateFS, "templates/devcontainer.json.tmpl"))

// DevContainer renders the devcontainer.json configuration to w. The output
// is a static JSON document that wires together the Dockerfile, firewall
// scripts, and Claude settings produced by other render functions. The cfg
// parameter is accepted for API consistency with other render functions,
// even though the current template contains no dynamic actions.
func DevContainer(w io.Writer, cfg GenerationConfig) error {
	if err := devcontainerTmpl.Execute(w, cfg); err != nil {
		return fmt.Errorf("render devcontainer.json: %w", err)
	}

	return nil
}
