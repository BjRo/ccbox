package render

import (
	"fmt"
	"io"
	"text/template"
)

var miseConfigTmpl = template.Must(template.ParseFS(templateFS, "templates/mise-config.toml.tmpl"))

// MiseConfig renders the mise-config.toml template to w. The output is a TOML
// configuration file for the mise runtime manager, listing all runtime tools
// and their versions. Node is expected to be present in cfg.Runtimes (ensured
// by EnsureNode at the call site).
func MiseConfig(w io.Writer, cfg GenerationConfig) error {
	if err := miseConfigTmpl.ExecuteTemplate(w, "mise-config.toml.tmpl", cfg); err != nil {
		return fmt.Errorf("render mise-config.toml: %w", err)
	}
	return nil
}
