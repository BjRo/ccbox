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

## Sentinel Error Mapping

When wrapping a third-party library, map its error values to application-level sentinel errors to avoid leaking library types across package boundaries:

- Define `var ErrXxx = errors.New("pkg: description")` in the package that owns the abstraction.
- Catch the library error with `errors.Is(err, libpkg.ErrSpecific)` and return the application sentinel instead.
- Callers use `errors.Is(err, yourpkg.ErrXxx)` without importing the library.
- Example: `huh.ErrUserAborted` is caught in `internal/wizard` and mapped to `wizard.ErrAborted`.

## TTY Detection for Interactive Features

Gate interactive behavior (wizards, prompts) on terminal detection:

- Use `golang.org/x/term.IsTerminal(int(f.Fd()))` via an unexported `isTerminal(r io.Reader) bool` helper.
- Type-assert `r` to `*os.File` first; non-file readers (pipes, `strings.Reader`) return false.
- When not a TTY, fall through to non-interactive behavior (auto-detect, flag-only mode).
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

## Package Documentation

When a package's doc comment outgrows a single line, extract it to a `doc.go` file. The original `.go` file keeps a bare `package <name>` line.
