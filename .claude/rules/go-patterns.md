---
description: Go coding patterns for agentbox — Cobra CLI, stdlib preferences, registry pattern, package conventions
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
- `var version = "dev"` in `cmd/root.go`, overridden at build time via `-ldflags "-X github.com/bjro/agentbox/cmd.version=..."`.
- GoReleaser sets this automatically. `go install` from source falls back to `"dev"`.

## Cobra Flag Conventions

**Singular nouns for multi-value flags**: Use `--stack` not `--stacks` for flags that accept comma-separated values.

**`trimAndFilter` for `StringSliceVar` flags**: Cobra's `StringSliceVar` preserves surrounding whitespace. Always trim and filter flag values before use.

**`resolveDir` pattern for `--dir` flags**: Empty means `os.Getwd()`, non-empty means `filepath.Abs()` + `os.Stat()` validation.

**Key=value pair flags via `StringSliceVar`**: For flags that accept `key=value` pairs (e.g., `--runtime-version go=1.22,node=20`), parse with `strings.SplitN(pair, "=", 2)` and validate both key and value are non-empty. Unknown keys that do not match any config entry are silently ignored (no coupling to the registry).

**Early validation against registries**: Validate flag values referencing registry entries immediately after parsing with a clear error listing valid options.

## Version Override Layering

When multiple sources can set the same configuration value (e.g., registry defaults, wizard choices, CLI flags), apply them in precedence order using a merge map:

1. Initialize from the lowest-precedence source (registry defaults via `Merge`).
2. Layer wizard/interactive choices on top.
3. Layer CLI flags on top (highest precedence for scripted use).
4. Apply the merged map to the config struct.

This pattern keeps each source independent and makes precedence explicit. See `cmd/init.go` for the `--runtime-version` flag layered over wizard `RuntimeVersions`.

## Parallel Slice for huh Form Values

Go maps do not support taking the address of a value (`&m[key]` is invalid). When `huh.NewInput().Value()` needs a `*string` pointer for each entry in a map, use a parallel slice:

```go
tools := slices.Sorted(maps.Keys(defaults))
values := make([]string, len(tools))
for i, tool := range tools {
    values[i] = defaults[tool]
}
// Build huh fields with &values[i], then transfer back to map.
```

Sort the keys first for deterministic form ordering.

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
    return fmt.Errorf(".devcontainer/ already exists in %s; run 'agentbox update' to regenerate, or remove it first", targetDir)
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

## Post-Merge Invariant Helpers

Container-build invariants (e.g., "node must always be present") are enforced by explicit helpers called after `render.Merge`, not inside `Merge` itself. This keeps `Merge` as a pure reflection of registry data for the selected stacks. The helper pattern:

- Takes `*GenerationConfig` (pointer, since it mutates).
- Is idempotent (no-op when the invariant already holds).
- Re-sorts after appending to maintain sorted-slice invariants.
- Is called after `Merge` and after version overrides, so user customizations are preserved.

See `render.EnsureNode` and ADR-0008.

## Shared Pure Helpers for Multi-Command Rendering

When multiple commands (`init`, `update`) need to produce the same set of rendered files, extract the shared logic into a package-level pure function in the `cmd` package:

- **Pure data arguments**: Accept `[]stack.StackID`, `[]string`, `map[string]string` -- not Cobra or wizard types.
- **Returns `map[string][]byte`**: Filename to content, no file I/O inside the function.
- **Caller assembles final content**: The helper returns the agentbox-managed portion only. Callers append command-specific content (e.g., `init` appends a fresh custom stage stub; `update` preserves the existing user stage).
- **No side effects**: Merge, EnsureNode, version overrides, and all template renders happen inside the helper. File writes and chmod happen in the caller.

See `renderFiles` in `cmd/init.go`, shared by `cmd/update.go`.

## Update Command: Config-Driven Regeneration

The `update` command reads `.agentbox.yml` for stacks and extra domains rather than re-detecting or prompting. This makes regeneration deterministic and scriptable.

- **`.agentbox.yml` as source of truth**: Stacks and domains default from the config file; `--stack` and `--extra-domains` flags override and persist back.
- **Preserve user content structurally**: Dockerfile is split at a structural boundary (`FROM agentbox AS custom`); everything from that line onward is preserved verbatim. `config.toml` is preserved entirely.
- **No `--runtime-version` on update**: Users edit `config.toml` directly for version changes.
- **`--force` recovery flag**: When the structural boundary is missing (e.g., user deleted the custom stage line), `--force` regenerates a fresh custom stage stub instead of erroring.
- **Sentinel errors for parsing**: `dockerfile.ErrNoCustomStage` is returned when the boundary is not found, allowing the caller to distinguish "missing marker" from other parse errors and decide whether `--force` applies.

## Package Documentation

When a package's doc comment outgrows a single line, extract it to a `doc.go` file. The original `.go` file keeps a bare `package <name>` line.
