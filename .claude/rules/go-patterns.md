---
description: Go coding patterns for ccbox — Cobra CLI, stdlib preferences, registry pattern, package conventions
globs: "**/*.go"
---

# Go Coding Patterns

## Cobra CLI

All commands follow the **unexported constructor pattern** established in `cmd/root.go`:

- Each command file exposes `newXxxCmd() *cobra.Command` (unexported).
- `newRootCmd()` builds the full command tree by calling sub-command constructors and wiring them via `AddCommand`.
- The package-level `var rootCmd = newRootCmd()` is the single production instance. No `init()` functions for command registration.
- Tests call `newRootCmd()` per test to get a fresh, isolated command tree. Use `cmd.SetOut()`, `cmd.SetErr()`, and `cmd.SetArgs()` for test I/O.
- Tests live in `package cmd` (internal), not `package cmd_test`, because they need access to unexported constructors.

Root command wiring:
- `SilenceErrors: true` -- Cobra does not print errors; `main.go` handles all error output.
- `SilenceUsage: true` -- Cobra does not dump usage on errors; users run `--help` explicitly.
- `main.go` prints the error to stderr and exits with code 1.

Version injection:
- `var version = "dev"` in `cmd/root.go`, overridden at build time via `-ldflags "-X github.com/bjro/ccbox/cmd.version=..."`.
- GoReleaser sets this automatically. `go install` from source falls back to `"dev"`.

## Cobra Flag Conventions

**Singular nouns for multi-value flags**: Use `--stack` not `--stacks` for flags that accept comma-separated values. This matches Go CLI conventions (e.g., `go build -tags`, `docker run --volume`). The comma-separated format `--stack go,node` reads more naturally with singular nouns.

**`trimAndFilter` for `StringSliceVar` flags**: Cobra's `StringSliceVar` splits on commas but preserves surrounding whitespace. Always trim and filter flag values before use:

```go
func trimAndFilter(values []string) []string {
    var result []string
    for _, v := range values {
        v = strings.TrimSpace(v)
        if v != "" {
            result = append(result, v)
        }
    }
    return result
}
```

This handles both `--stack "go, node"` (spaces after commas) and `--stack "go,,node"` (empty elements).

**`resolveDir` pattern for `--dir` flags**: Directory flags follow a standard resolution pattern: empty means `os.Getwd()`, non-empty means `filepath.Abs()` + `os.Stat()` validation (exists and is a directory). Extract this into a named helper (`resolveDir`) rather than inlining in `RunE`.

**Early validation against registries**: When a flag value references registry entries (e.g., stack IDs), validate immediately after flag parsing with a clear error listing valid options. Do not defer validation to downstream functions that may produce less helpful error messages. Build the valid-options string lazily (only on error) to avoid allocation in the happy path.

**No-op flags for API contracts**: When a future feature has a known CLI interface (e.g., `--non-interactive` for a wizard that does not exist yet), add the flag now as a no-op with `_ = flagVar` to suppress the unused warning. This establishes the contract so scripts can be written before the feature ships, and prevents flag-name bikeshedding later.

## Prefer Modern stdlib Packages

Since the project targets Go 1.24+, prefer `slices` and `maps` from the standard library:

- `slices.Sort(s)` or `slices.SortFunc(s, cmp)` instead of `sort.Slice(s, less)`.
- `slices.Sorted(maps.Keys(m))` instead of manually collecting/sorting keys.
- `slices.Clone(s)` instead of manual `make` + `copy`.
- `maps.Clone(m)` for shallow copies.

## Prefer `default` in Category/Enum Switches

When switching on a string-typed category or enum where one branch is the "safe" fallback, use `default` instead of explicitly listing all non-primary cases. This prevents silent data loss if a new category value is added.

## Registry Pattern

Packages that own static lookup data (`internal/stack/`, `internal/firewall/`) follow:

- **Unexported `var registry` map**: Public API via accessor functions only.
- **Defensive-copy accessors**: `All() []T`, `Get(id) (T, bool)`, `IDs() []ID` all return deep copies via `slices.Clone`.
- **String-based type IDs**: Use `type FooID string` when IDs appear in config files, CLI flags, or template output.
- **Sorted output**: `All()` and `IDs()` return sorted slices for deterministic output.
- **`init()` acceptable for static data**: The `init()` prohibition applies to command registration, not immutable data initialization.

## Package Documentation

When a package's doc comment outgrows a single line, extract it to a `doc.go` file. The original `.go` file keeps a bare `package <name>` line.
