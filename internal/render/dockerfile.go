package render

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed templates/Dockerfile.tmpl
var dockerfileTemplate string

// Dockerfile renders the Dockerfile template using the given GenerationConfig.
// It returns the rendered content as a string.
func Dockerfile(cfg GenerationConfig) (string, error) {
	tmpl, err := template.New("Dockerfile").Parse(dockerfileTemplate)
	if err != nil {
		return "", fmt.Errorf("render: parse Dockerfile template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, cfg); err != nil {
		return "", fmt.Errorf("render: execute Dockerfile template: %w", err)
	}

	return buf.String(), nil
}
