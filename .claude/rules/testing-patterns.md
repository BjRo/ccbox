---
description: Testing strategies — fs.FS testability, registry-backed assertions, template output testing, interface fakes, CLI test isolation
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

## Interface-Based Test Doubles

When a dependency cannot be unit-tested (terminal I/O, network), define a narrow interface and inject via constructor parameters:

- **Interface in the owning package**: e.g., `wizard.Prompter` with a single `Run(detected) (Choices, error)` method.
- **Fake in test files**: Struct with canned return values. Add `failIfCalled` guard to assert code paths that must NOT invoke the dependency.
- **Nil means default**: Constructor accepts the interface; `nil` triggers real implementation.

## CLI Test Directory Isolation

- **Prefer `--dir` flag over `os.Chdir()`** for parallel test execution.
- **Use `t.Chdir()` (Go 1.24+)** when testing the "no --dir means current directory" fallback.
- **Use `t.TempDir()`** for output directories.

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
- **Anchor assertions to rendered structure**: When checking for short tokens (e.g., stack ID `"go"`), assert against the rendered format (`"- go\n"` for a list item, `"| go |"` for a table cell) rather than bare `strings.Contains(out, "go")`. Bare substring matches produce false positives.

## JSON Template Testing

Templates that produce JSON require targeted validation:

- **Unmarshal validity**: `json.Unmarshal` the rendered output into a typed struct. This catches trailing commas, unescaped quotes, and malformed arrays.
- **Raw array form**: Use `json.RawMessage` to verify empty arrays render as `[]` not `null`.
- **Special character round-trip**: Test with strings containing `"`, `\`, and control characters to verify the `jsonString` FuncMap helper produces valid JSON that round-trips correctly through marshal/unmarshal.
- **Static template verification**: When a template has no Go template actions, render with different configs and assert byte-equality to prove it is truly stack-agnostic.

## YAML Serialization Testing

Packages that marshal/unmarshal YAML (`internal/config/`) use round-trip verification:

- **Write-then-Load round-trip**: Create a struct, write it to a `bytes.Buffer`, load it back, verify all fields match. This is the primary correctness test.
- **Format spot-checks**: Write to a buffer and assert expected YAML strings appear (`version: 1`, `stacks: [go, node]`). Do not assert exact full output -- timestamps and field ordering may vary.
- **Empty vs nil slices**: Verify both `nil` and `[]T{}` inputs render as `[]` (not `null`). Verify omitted fields decode to non-nil empty slices.
- **Schema version validation**: Test that `Load` rejects unknown version numbers with a clear error.
- **Timestamp precision**: `yaml.v3` truncates `time.Time` to second precision. Test round-trips with second-level precision only; do not rely on sub-second accuracy.
