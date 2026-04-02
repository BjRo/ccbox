# ADR-0003: Separate data registries from behavior packages

- **Date**: 2026-04-02
- **Status**: Accepted
- **Bean**: ccbox-d39x

## Context

The stack metadata registry holds static data (runtime versions, LSP servers, domain allowlists, marker files) that is consumed by multiple packages: `detect` (stack scanning), `firewall` (domain allowlists), and `render` (template rendering). Placing the registry inside any one consumer would create an import cycle when the others need it.

## Decision

Data registries live in their own package (`internal/stack/`), separate from the packages that implement behavior on top of that data. The registry package exports only types and read-only accessors (`Get`, `All`, `IDs`). Behavior packages import the registry but the registry never imports behavior packages.

Design constraints for registry packages:

- **Composite literal initialization**: Use `var registry = map[...]...{}` at package level, not `init()` functions. This keeps initialization explicit and inspectable.
- **Defensive deep copies**: All accessor functions must return copies of structs containing slices or maps. Use `slices.Clone` for slice fields so callers cannot mutate internal state.
- **Sorted output**: Functions that return collections (`All`, `IDs`) sort results for deterministic output in templates and CLI displays.
- **String-typed identifiers**: Use `type FooID string` rather than integer enums when IDs appear in config files, CLI flags, or template output, avoiding a marshaling layer.

## Consequences

- No import cycles between data and behavior packages.
- Adding a new stack requires editing only `internal/stack/stack.go`; consumer packages pick it up automatically.
- Every new registry in the project should follow the same pattern (composite literal, defensive copies, sorted accessors).
- Registry tests should cover: completeness, copy isolation (mutation does not leak), sort order, and data integrity invariants (no duplicate marker files, valid hostnames).
