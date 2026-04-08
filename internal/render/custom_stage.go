package render

import (
	"bytes"
	"fmt"
	"text/template"
)

var customStageTmpl = template.Must(template.ParseFS(templateFS, "templates/custom-stage.tmpl"))

// CustomStage renders the custom stage stub template. The template is static
// (no GenerationConfig needed) and produces the FROM line plus helpful
// comments for user customizations.
func CustomStage() (string, error) {
	var buf bytes.Buffer
	if err := customStageTmpl.ExecuteTemplate(&buf, "custom-stage.tmpl", nil); err != nil {
		return "", fmt.Errorf("render custom stage: %w", err)
	}
	return buf.String(), nil
}
