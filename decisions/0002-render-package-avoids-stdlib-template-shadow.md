# ADR-0002: Use `render` package name instead of `template` to avoid stdlib shadow

- **Date**: 2026-04-02
- **Status**: Accepted
- **Bean**: ccbox-5333

## Context

The template rendering package under `internal/` was initially planned as `internal/template/`. This package will heavily import `text/template` from the standard library. Having a local package named `template` forces every file that uses both the local package and the stdlib to alias one of them, creating friction and inconsistency across the codebase.

## Decision

Name the package `internal/render/` instead of `internal/template/`. The name `render` describes the action the package performs (rendering templates into devcontainer configuration files) and avoids shadowing `text/template` and `html/template`.

## Consequences

- No import aliasing needed anywhere in the codebase when using both `render` and `text/template`.
- The package reads naturally at call sites: `render.DevContainer(...)`.
- Downstream beans that add template rendering logic land in `internal/render/`.
