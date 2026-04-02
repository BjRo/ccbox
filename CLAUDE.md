# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**ccbox** is a Go CLI tool that generates `.devcontainer/` setups for running Claude Code in sandboxed environments with full permissions and network isolation. It auto-detects tech stacks (Go, Node/TypeScript, Python, Rust, Ruby), generates Dockerfiles, devcontainer.json, firewall scripts, and Claude Code settings. Distributed via Homebrew and GitHub Releases.

The project is in early development. All planned work is tracked as beans (see `.beans/`). The root milestone is `ccbox-el52` (v0.1.0 MVP Release).

## Build & Development Commands

```bash
go build ./...                  # Build
go test ./...                   # Run all tests
go test ./internal/detect/...   # Run tests for a specific package
go test -run TestName ./...     # Run a single test
golangci-lint run ./...         # Lint
```

- **Go version**: 1.24+
- **Module path**: `github.com/bjro/ccbox`
- **Release tooling**: GoReleaser (cross-platform: linux/darwin, amd64/arm64), Homebrew tap at `bjro/homebrew-tap`

## Architecture

```
cmd/
  root.go              # Root Cobra command
  init.go              # `ccbox init` subcommand (interactive wizard + CLI flags)
internal/
  stack/               # Stack metadata registry (pure data, zero internal dependencies)
  detect/              # Stack detection (scans for marker files like go.mod, package.json, etc.)
  render/              # Template rendering engine (Go templates → Dockerfile, devcontainer.json, scripts)
  firewall/            # Domain allowlist logic (per-stack defaults, merging, deduplication)
  config/              # .ccbox.yml handling (persists user choices)
main.go
```

Key design patterns:
- **Stack metadata registry**: single source of truth per stack (runtime versions, LSP servers, default domains). Data lives in `internal/stack/`, separate from behavior packages (`detect`, `firewall`, `render`) to avoid import cycles. See ADR-0003.
- **Multi-stack merging**: projects with multiple stacks get merged configurations
- **Dual-mode UX**: interactive wizard (default) and non-interactive CLI flags (`--stacks=go,node --domains=...`)
- Templates use Go's `embed` package for bundling

## Registry Pattern

Packages that own static lookup data (`internal/stack/`, `internal/firewall/`) follow a consistent pattern:

- **Unexported `var registry` map**: Public API via accessor functions only. No exported map or mutable state.
- **Defensive-copy accessors**: `All() []T`, `Get(id) (T, bool)`, `IDs() []ID` all return deep copies. Slice fields cloned via `slices.Clone`. Tests verify mutations don't corrupt canonical data.
- **String-based type IDs**: Use `type FooID string` (not integer enums) when IDs appear in config files, CLI flags, or template output.
- **Sorted output**: `All()` and `IDs()` return sorted slices via `slices.Sorted(maps.Keys(m))` for deterministic templates and CLI output.
- **`init()` acceptable for static data**: The `init()` prohibition in Cobra CLI Patterns applies to command registration. Package-level initialization of static, immutable data is idiomatic Go.

## Testing Patterns for Registry-Backed Code

When testing functions that consume registry data (e.g., `firewall.Merge`), prefer **structural invariants computed from the registry** over hardcoded expected values. Hardcoded counts break silently when registry data grows. Pair structural assertions with a few **hardcoded spot-checks** that name specific well-known entries, so the two approaches cross-validate each other.

Example: assert `len(result.Static) == len(collectExpected(...))` (structural), then `assert result contains "github.com" in Static` (spot-check).

## Bean-Driven Workflow

All work is tracked with `beans` CLI, not TodoWrite. The delivery pipeline:

1. **`/refine <bean-id>`** -- Create detailed implementation plan
2. **`/challenge <bean-id>`** -- Stress-test plan via Go engineer persona
3. **`/implement <bean-id>`** -- TDD-based implementation
4. **`/rework`** -- Fix review feedback
5. **`/codify <bean-id>`** -- Extract learnings into docs/ADRs
6. **`/deliver <bean-id>`** -- Run full pipeline end-to-end

Use `/dev-workflow` when starting work on a bean for proper git hygiene.

## Git Conventions

- **Branch naming**: `<type>/<bean-id>-<slug>` (e.g., `feat/ccbox-5333-initialize-go-module`)
  - `feat/` for features, `fix/` for bugs, `chore/` for tasks
- Use `.claude/scripts/start-work.sh <bean-id>` to create branches with correct naming
- Hooks enforce: branch naming validation, bean checklist completion before marking done, .env file access blocking
- Always include updated bean files in commits alongside code changes

## Definition of Done

Every bean requires: TDD tests, no TODO/FIXME/HACK/XXX comments, lint passing, tests passing, PR created, automated code review via `@review-backend`, review feedback addressed, ADR written if architectural changes.

## Code Review Standards

Automated reviews use the Go engineer persona (`.claude/personas/go-engineer.md`) with severity levels:
- **CRITICAL**: Must fix (security, correctness, data loss)
- **WARNING**: Should fix (design, maintainability)
- **SUGGESTION**: Consider (style, minor improvements)

Engineering calibration: flag repetition (DRY), flag over-engineering (premature abstractions), flag under-engineering (missing error handling/edge cases).

## Cobra CLI Patterns

All commands follow the **unexported constructor pattern** established in `cmd/root.go`:

- Each command file exposes `newXxxCmd() *cobra.Command` (unexported).
- `newRootCmd()` builds the full command tree by calling sub-command constructors and wiring them via `AddCommand`.
- The package-level `var rootCmd = newRootCmd()` is the single production instance. No `init()` functions for command registration.
- Tests call `newRootCmd()` per test to get a fresh, isolated command tree. Use `cmd.SetOut()`, `cmd.SetErr()`, and `cmd.SetArgs()` for test I/O.
- Tests live in `package cmd` (internal), not `package cmd_test`, because they need access to unexported constructors. True black-box CLI testing belongs in integration tests.

Root command wiring:
- `SilenceErrors: true` -- Cobra does not print errors; `main.go` handles all error output.
- `SilenceUsage: true` -- Cobra does not dump usage on errors; users run `--help` explicitly.
- `main.go` prints the error to stderr and exits with code 1.

Version injection:
- `var version = "dev"` in `cmd/root.go`, overridden at build time via `-ldflags "-X github.com/bjro/ccbox/cmd.version=..."`.
- GoReleaser sets this automatically. `go install` from source falls back to `"dev"`.

## Go Style: Prefer Modern stdlib Packages

Since the project targets Go 1.24+, prefer the `slices` and `maps` packages from the standard library over older patterns:

- **Sorting**: `slices.Sort(s)` or `slices.SortFunc(s, cmp)` instead of `sort.Slice(s, less)`.
- **Sorted map keys**: `slices.Sorted(maps.Keys(m))` instead of manually collecting keys, sorting, and returning.
- **Slice copying**: `slices.Clone(s)` instead of manual `make` + `copy`.
- **Map copying**: `maps.Clone(m)` for shallow copies.

These produce shorter, less error-prone code and signal to readers that the codebase follows current Go idioms.

## Go Style: Prefer `default` in Category/Enum Switches

When switching on a string-typed category or enum where one branch is the "safe" fallback, use `default` instead of explicitly listing all non-primary cases. This prevents silent data loss if a new category value is added to the type but not yet handled in the switch. For example, `firewall.Merge` routes unrecognized `Category` values to the Dynamic bucket (re-resolved by dnsmasq) rather than silently dropping them.

## Linting

- Config: `.golangci.yml` using golangci-lint **v2** format (`version: "2"`).
- Enabled linters: govet, errcheck, staticcheck, unused, ineffassign.
- Run with `golangci-lint run ./...`.

## Decisions

Architecture Decision Records live in `decisions/`. Create new ones with `/decision` when introducing dependencies, patterns, or architectural changes.
