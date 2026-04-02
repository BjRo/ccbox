// Package render merges detected stack metadata into a single GenerationConfig
// and renders Go templates into devcontainer configuration files. The Merge
// function is the entry point: it combines runtime versions, language servers,
// and domain allowlists from multiple stacks into a deduplicated, sorted
// configuration ready for template rendering.
package render
