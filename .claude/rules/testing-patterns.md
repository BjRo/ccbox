---
description: Testing strategies — fs.FS testability, registry-backed assertions, template output testing
globs: "**/*_test.go"
---

# Testing Patterns

## Filesystem Testability via fs.FS

Packages that perform filesystem I/O should use Go's `fs.FS` interface:

- **Unexported core function accepts `fs.FS`**: e.g., `detect(fsys fs.FS) ([]stack.StackID, error)`.
- **Exported function accepts a path string**: Validates path, wraps with `os.DirFS(dir)`, delegates to core.
- **Tests use `fstest.MapFS`**: In-memory filesystem, zero disk I/O, deterministic.
- Use `fs.Stat`, `fs.ReadDir`, `fs.Glob` (not `os.*` or `filepath.*`) in the core function.
- Reserve one integration-style test that calls the public API with a real path.

## Registry-Backed Code

Prefer **structural invariants computed from the registry** over hardcoded expected values. Hardcoded counts break silently when registry data grows. Pair structural assertions with **hardcoded spot-checks** for well-known entries.

Example: `len(result.Static) == len(collectExpected(...))` (structural) + `result contains "github.com" in Static` (spot-check).

## Template Testing

Use **structural assertions**, not golden-file snapshots:

- **Two-tier strategy**: Integration tests (through `Merge` + render) for full pipeline; isolation tests (hand-built `GenerationConfig`) for template logic independent of registry.
- **Registry-computed completeness**: Iterate `cfg.Domains.Static` and assert each domain appears in output.
- **Spot-checks**: Assert well-known entries appear in output.
- **Empty-input safety**: Render with empty (non-nil) slices, verify no `<no value>` artifacts.
- **Shell syntax validation**: Assert no bare backslash lines, no double backslashes, no blank lines inside RUN blocks.
- **Defense-layer verification**: Assert single-quoted domain interpolation in shell script output.

## JSON Template Testing

Templates that produce JSON require targeted validation:

- **Unmarshal validity**: `json.Unmarshal` the rendered output into a typed struct. This catches trailing commas, unescaped quotes, and malformed arrays.
- **Raw array form**: Use `json.RawMessage` to verify empty arrays render as `[]` not `null`.
- **Special character round-trip**: Test with strings containing `"`, `\`, and control characters to verify the `jsonString` FuncMap helper produces valid JSON that round-trips correctly through marshal/unmarshal.
- **Static template verification**: When a template has no Go template actions, render with different configs and assert byte-equality to prove it is truly stack-agnostic.
