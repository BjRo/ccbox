# ADR-0008: EnsureNode as post-Merge invariant enforcement

- **Date**: 2026-04-07
- **Status**: Accepted
- **Bean**: agentbox-dn7n

## Context

Claude Code requires Node/npm at runtime, so the generated container must always include a `node` runtime entry in its mise configuration. Previously, this invariant was enforced inside the Dockerfile template itself: `node = "lts"` was hardcoded in an inline COPY heredoc, and `{{ if ne .Tool "node" }}` skipped node in the runtime loop to avoid duplication.

When we extracted the mise config to a standalone `config.toml` file and added user-configurable runtime versions, we needed a new place to enforce the "node is always present" invariant. The options were:

1. **Inject node inside `render.Merge`** -- `Merge` would always append a node runtime to the merged `GenerationConfig`.
2. **Inject node at the orchestration layer** -- A separate helper called from `cmd/init.go` after `Merge` returns and after version overrides are applied.

## Decision

We chose option 2: a separate `render.EnsureNode(cfg *GenerationConfig)` function called from `cmd/init.go`.

`render.Merge` remains a pure reflection of registry data for the selected stacks. It does not inject entries that are not backed by a user-selected stack. The "node must always be present" rule is a container-build invariant (Claude Code needs npm), not a registry-level truth. `EnsureNode` encodes this invariant as an explicit, testable step at the orchestration layer.

`EnsureNode` is called after `Merge` returns and after version overrides are applied. This ordering matters: if the user selected the Node stack and overrode its version via wizard or CLI flag, `EnsureNode` sees the already-customized node runtime and is a no-op. If no stack includes node, `EnsureNode` appends `{Tool: "node", Version: "lts"}` and re-sorts to maintain the Tool-sorted invariant.

## Consequences

- `render.Merge` stays pure and its tests do not need to account for injected node entries.
- The "node is always present" invariant is tested in isolation via `ensure_test.go` (7 test cases including idempotency, sort preservation, nil/empty handling).
- The Dockerfile template no longer special-cases node -- the `mise-config.toml.tmpl` iterates all runtimes uniformly.
- Any future "always-present" runtime (if one arises) would follow the same pattern: a post-Merge helper in `internal/render/`, called from the orchestration layer.
