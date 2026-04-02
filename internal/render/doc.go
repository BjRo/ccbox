// Package render merges detected stack metadata into a single GenerationConfig
// and renders embedded Go templates into devcontainer configuration files.
//
// The [Merge] function is the entry point: it combines runtime versions,
// language servers, and domain allowlists from multiple stacks into a
// deduplicated, sorted [GenerationConfig] ready for template rendering.
//
// Template rendering functions ([RenderFirewall]) execute embedded
// text/template files from the templates/ subdirectory against a
// [GenerationConfig] and return rendered bytes. They perform no file I/O;
// actual file writing is the responsibility of the orchestrator.
package render
