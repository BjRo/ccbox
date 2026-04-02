// Package detect scans project directories for marker files to identify
// which technology stacks are in use. It checks the project root and one
// level of subdirectories, skipping well-known noise directories like
// vendor, node_modules, and .git.
//
// Detection uses two mechanisms: exact filename matching against the
// MarkerFiles registered in the stack registry, and glob-pattern matching
// for stacks that use wildcard markers (e.g., *.gemspec for Ruby).
// Glob patterns live in this package because the stack registry
// deliberately stores only exact filenames.
package detect
