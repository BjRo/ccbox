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
- **Linting**: `.golangci.yml` using golangci-lint **v2** format. Enabled: govet, errcheck, staticcheck, unused, ineffassign.
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
  firewall/            # Domain allowlist logic (per-stack defaults, merging, deduplication, validation)
  config/              # .ccbox.yml handling (persists user choices)
main.go
```

Key design:
- **Stack metadata registry** in `internal/stack/` -- single source of truth per stack, separate from behavior packages to avoid import cycles (ADR-0004)
- **`render.Merge`** -- single entry point for multi-stack merging into `GenerationConfig` (ADR-0005)
- **Embedded templates** in `internal/render/templates/`, parsed once at startup (ADR-0006)
- **Dual-mode UX** -- interactive wizard (default) and non-interactive CLI flags

## Bean-Driven Workflow

All work is tracked with `beans` CLI, not TodoWrite. The delivery pipeline:

1. **`/refine <bean-id>`** -- Create detailed implementation plan
2. **`/challenge <bean-id>`** -- Stress-test plan via Go engineer persona
3. **`/implement <bean-id>`** -- TDD-based implementation
4. **`/rework`** -- Fix review feedback
5. **`/codify <bean-id>`** -- Extract learnings into rules/ADRs
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

## Decisions

All important technical decisions are documented as Architecture Decision Records (ADRs). See [`decisions/README.md`](decisions/README.md) for the full index. Create new ones with `/decision` when introducing dependencies, patterns, or architectural changes.

## Rules

Go coding patterns, template rendering conventions, and testing strategies live in `.claude/rules/`. These are loaded automatically by Claude Code. When codifying learnings from completed work, prefer adding to rules over expanding this file.
