# ADR-0005: render.Merge as single merge orchestrator

- **Date**: 2026-04-02
- **Status**: Accepted
- **Bean**: ccbox-tv1t

## Context

When multiple tech stacks are detected in a project, their metadata (runtimes, LSP servers, domain allowlists) must be merged into a single configuration for template rendering. The merging logic touches data owned by two packages: `internal/stack/` (runtimes, LSPs) and `internal/firewall/` (domain allowlists). We needed to decide where the orchestration lives and how the merge responsibilities are split.

## Decision

`render.Merge` is the single entry point for producing a template-ready configuration. It owns the `GenerationConfig` struct -- the sole input to the template rendering pipeline. The function:

1. Validates all stack IDs against the stack registry (fail-fast on unknown IDs).
2. Deduplicates and sorts stack IDs.
3. Collects runtimes and LSPs from the stack registry, deduplicating by key (Tool for runtimes, Package for LSPs). When two stacks share a key, the alphabetically-first stack wins because stacks are sorted before collection.
4. Delegates domain merging to `firewall.Merge`, which owns the domain-specific logic (always-on domains, per-stack domains, user extras, Static/Dynamic categorization).
5. Ensures all output slices are non-nil (empty `[]T{}` instead of `nil`) for template safety.

`firewall.Merge` remains the authority on domain allowlist merging. It is called by `render.Merge` but is also independently testable and usable.

## Consequences

- Templates receive a single, fully-resolved `GenerationConfig` with no further merging needed. Template code stays simple.
- Adding a new field to the merged config (e.g., editor extensions) means extending `GenerationConfig` and adding a collection step in `render.Merge`. The pattern is uniform: deduplicate by key, sort, ensure non-nil.
- `render` imports both `stack` and `firewall`. This is intentional -- `render` is a leaf consumer that sits above both data packages in the dependency graph. Neither `stack` nor `firewall` imports `render`.
- Unknown stack IDs produce an error in `render.Merge` (strict validation), whereas `firewall.Merge` silently skips them (lenient). This is deliberate: `render.Merge` is the user-facing entry point where early validation prevents confusing downstream failures.
