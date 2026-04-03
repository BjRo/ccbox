---
description: Go coding patterns for ccbox — Cobra CLI, stdlib preferences, registry pattern, package conventions
globs: "**/*.go"
---

# Go Coding Patterns

## Cobra CLI

All commands follow the **unexported constructor pattern** established in `cmd/root.go`:

- Each command file exposes `newXxxCmd(deps) *cobra.Command` (unexported). Parameters are acceptable when the command needs injectable dependencies (e.g., `newInitCmd(prompter wizard.Prompter)`).
- `newRootCmd(deps)` builds the full command tree by calling sub-command constructors and wiring them via `AddCommand`. Production call passes `nil` for optional dependencies; constructors fall back to real implementations internally.
- The package-level `var rootCmd = newRootCmd(nil)` is the single production instance. No `init()` functions for command registration.
- Tests call `newRootCmd(fake)` per test to get a fresh, isolated command tree with injected fakes. Use `cmd.SetOut()`, `cmd.SetErr()`, and `cmd.SetArgs()` for test I/O.
- Tests live in `package cmd` (internal), not `package cmd_test`, because they need access to unexported constructors.
- **Prefer parameter injection over package-level function variables** for test seams. Function variables are shared mutable state that breaks `t.Parallel()`. Constructor parameters keep tests isolated.

Root command wiring:
- `SilenceErrors: true` -- Cobra does not print errors; `main.go` handles all error output.
- `SilenceUsage: true` -- Cobra does not dump usage on errors; users run `--help` explicitly.
- `main.go` prints the error to stderr and exits with code 1.

Version injection:
- `var version = "dev"` in `cmd/root.go`, overridden at build time via `-ldflags "-X github.com/bjro/ccbox/cmd.version=..."`.
- GoReleaser sets this automatically. `go install` from source falls back to `"dev"`.

## Cobra Flag Conventions

**Singular nouns for multi-value flags**: Use `--stack` not `--stacks` for flags that accept comma-separated values.

**`trimAndFilter` for `StringSliceVar` flags**: Cobra's `StringSliceVar` preserves surrounding whitespace. Always trim and filter flag values before use.

**`resolveDir` pattern for `--dir` flags**: Empty means `os.Getwd()`, non-empty means `filepath.Abs()` + `os.Stat()` validation.

**Early validation against registries**: Validate flag values referencing registry entries immediately after parsing with a clear error listing valid options.

## Sentinel Error Mapping

Map third-party library error values to application-level sentinel errors:

- Define `var ErrXxx = errors.New("pkg: description")` in the owning package.
- Catch library errors with `errors.Is` and return the application sentinel.
- Example: `huh.ErrUserAborted` → `wizard.ErrAborted`.

## TTY Detection for Interactive Features

- Use `golang.org/x/term.IsTerminal(int(f.Fd()))` via `isTerminal(r io.Reader) bool` helper.
- Type-assert `r` to `*os.File` first; non-file readers return false.
- When not a TTY, fall through to non-interactive behavior.
- When an injected dependency (e.g., `Prompter`) is non-nil, use it regardless of TTY state (test path).

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

## Non-nil Empty Slices for Serialization

Slices that will be serialized (YAML, JSON, or Go templates) must be initialized as `[]T{}` rather than left as `nil`. This applies in two places:

- **Before marshaling**: Defensive nil-to-empty conversion prevents `null` output. For `yaml.v3`, pair this with the `,flow` struct tag to render `[]` instead of a block-style empty sequence.
- **After unmarshaling**: Fields omitted in the source document decode as `nil`. Normalize to `[]T{}` after decode for consistent downstream behavior.

The `internal/render` package applies this for Go templates; `internal/config` applies it for YAML. The principle is the same: callers should never need to distinguish nil from empty.

## Serialization Boundary Packages

Packages that sit at a serialization boundary (`internal/config/` for YAML, future API clients, etc.) use **primitive types only** in their exported structs -- `string`, `int`, `time.Time`, `[]string` -- not domain types like `stack.StackID`. The `cmd` layer converts between domain types and primitives. This keeps the serialization package free of imports from other internal packages and avoids import cycles.

## File Writing in Commands

The `cmd` layer writes files via `bytes.Buffer` + `os.WriteFile`, not `os.Create` + `defer Close`. This avoids two problems:

1. **Swallowed close errors**: `defer f.Close()` discards the error, which can hide filesystem failures (disk full, NFS errors).
2. **Consistency**: The `.devcontainer/` file writes already use `os.WriteFile`. New file writes should match.

When a package exposes a `Write(w io.Writer, ...)` function, the command renders to a `bytes.Buffer` first, then calls `os.WriteFile` with the buffer contents.

## Pre-existence Guards for Output Directories

When a command creates an output directory (e.g., `.devcontainer/`), check for conflicts before doing any work:

```go
if _, statErr := os.Stat(outDir); statErr == nil {
    return fmt.Errorf(".devcontainer/ already exists in %s; remove it first or use a different directory", targetDir)
}
```

- **Use `os.Stat` without `IsDir()`**: Treat any existing entry (file or directory) at the target path as a conflict. Checking only `IsDir()` lets a regular file slip through, causing an opaque `os.MkdirAll` failure downstream.
- **Fail before side effects**: Place the guard before any rendering, merging, or I/O so the command exits cleanly with no partial output.

## YAML with `gopkg.in/yaml.v3`

- Use `yaml.NewEncoder(w)` / `yaml.NewDecoder(r)` rather than `yaml.Marshal` / `yaml.Unmarshal` for streaming-friendly I/O.
- Call `enc.SetIndent(2)` for human-readable output.
- Call `enc.Close()` to flush the encoder's internal buffer -- this is easy to forget.
- `time.Time` fields marshal as unquoted YAML timestamps (e.g., `generated_at: 2026-04-02T10:00:00Z`). This is valid YAML and round-trips correctly. Do not force quoting.
- Use the `,flow` struct tag on slice fields to render `[]` for empties and `[a, b]` for short lists, instead of block-style sequences.

## Package Documentation

When a package's doc comment outgrows a single line, extract it to a `doc.go` file. The original `.go` file keeps a bare `package <name>` line.
