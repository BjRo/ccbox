package render

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

//go:embed templates/Dockerfile.tmpl
var dockerfileTemplate string

var dockerfileTmpl = template.Must(template.New("Dockerfile").Parse(dockerfileTemplate))

// Dockerfile renders the Dockerfile template using the given GenerationConfig.
// It returns the rendered content as a string.
func Dockerfile(cfg GenerationConfig) (string, error) {
	var buf bytes.Buffer
	if err := dockerfileTmpl.Execute(&buf, cfg); err != nil {
		return "", fmt.Errorf("render: execute Dockerfile template: %w", err)
	}

	return buf.String(), nil
}
