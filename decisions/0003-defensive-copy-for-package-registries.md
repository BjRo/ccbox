# ADR-0003: Defensive deep copy for package-level registries

- **Date**: 2026-04-02
- **Status**: Accepted
- **Bean**: ccbox-ztaa

## Context

The `internal/firewall/` package owns a curated domain allowlist stored in a package-level `var registry` map. The initial implementation returned shallow copies from `Registry()` and `ForStack()`: the map was copied, but each `Allowlist` value's `Domains` slice header still pointed at the original backing array. This meant callers could silently corrupt the canonical data by mutating slice elements (e.g., `al.Domains[0].Name = "evil.com"`).

This was caught during code review on PR #2. The existing test (`TestRegistry_ReturnsDefensiveCopy`) only verified map-key deletion, not slice-element mutation, giving false confidence that the copy was safe.

## Decision

All accessor functions on package-level registries must return deep copies of the data. Specifically:

1. Use a `copyAllowlist` helper that clones the `Domains` slice via `slices.Clone()`. Since `Domain` is a pure value type (no pointers, no nested slices), cloning the slice is sufficient for a deep copy.
2. Both `Registry()` and `ForStack()` call `copyAllowlist` before returning.
3. Tests must verify both map-key isolation (deleting a key from one copy does not affect another) and slice-element isolation (mutating an element in the returned slice does not affect the canonical data).

This pattern applies to any future registry in the codebase (e.g., stack metadata in `internal/detect/`).

## Consequences

- Callers can freely mutate returned data (append, filter, sort domains) without corrupting shared state. This is important for the downstream merging logic (`ccbox-ff1i`).
- Each call allocates a new map and cloned slices. This is negligible for small registries (6 stacks, ~17 total domains).
- If a future struct contains pointer fields or nested slices, `slices.Clone` alone will not suffice -- the copy helper must be extended to clone those fields explicitly.
- The defensive copy test pattern (mutate-then-verify) should be applied to all new registries.
