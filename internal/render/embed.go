package render

import "embed"

//go:embed templates/devcontainer.json.tmpl
var templateFS embed.FS
