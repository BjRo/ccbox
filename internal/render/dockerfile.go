package render

import (
	"bytes"
	"fmt"
	"text/template"
)

var dockerfileTmpl = template.Must(template.ParseFS(templateFS, "templates/Dockerfile.tmpl"))

// Dockerfile renders the Dockerfile template using the given GenerationConfig.
// It returns the rendered content as a string.
func Dockerfile(cfg GenerationConfig) (string, error) {
	var buf bytes.Buffer
	if err := dockerfileTmpl.ExecuteTemplate(&buf, "Dockerfile.tmpl", cfg); err != nil {
		return "", fmt.Errorf("render: execute Dockerfile template: %w", err)
	}

	return buf.String(), nil
}
