---
# agentbox-kyij
title: Refactor LSP Plugin field to be tool-agnostic
status: in-progress
type: task
priority: normal
created_at: 2026-04-08T09:15:46Z
updated_at: 2026-04-08T09:39:25Z
parent: agentbox-cqi5
---

Refactor `LSP.Plugin string` in `internal/stack/stack.go` to `LSP.Plugins map[string]string`, keyed by coding tool identifier. This decouples the stack registry from any single coding tool and prepares for Codex (and future tools) that may have their own plugin systems.

## Scope

### `internal/stack/stack.go`
- Replace `Plugin string` in `LSP` struct with `Plugins map[string]string`
- Define constant: `CodingToolClaude = "claude"`
- Update all 5 stack registry entries:
  - Go: `Plugins: map[string]string{"claude": "gopls-lsp@claude-plugins-official"}`
  - Node: `Plugins: map[string]string{"claude": "typescript-lsp@claude-plugins-official"}`
  - Python: `Plugins: map[string]string{"claude": "pyright-lsp@claude-plugins-official"}`
  - Rust: `Plugins: map[string]string{"claude": "rust-analyzer-lsp@claude-plugins-official"}`
  - Ruby: `Plugins: map[string]string{}` (empty map)
- Update `copyStack()` to deep-copy the Plugins map via `maps.Clone`

### `internal/render/claude.go` + `claude-user-settings.json.tmpl`
- Add `claudePlugin` FuncMap helper: extracts `"claude"` key from Plugins map
- Update template to use new helper: `{{range $i, $lsp := .LSPs}}{{$p := $lsp.Plugins | claudePlugin}}{{if $p}}...{{end}}{{end}}`

### `internal/render/render.go`
- No structural changes needed -- `LSP` struct change flows through `GenerationConfig.LSPs`

### Tests to update
- `internal/stack/stack_test.go` -- update for Plugins map
- `internal/render/claude_test.go` -- update for Plugins map, test new FuncMap helper
- `internal/render/render_test.go` -- update for Plugins map
- `internal/render/dockerfile_test.go` -- update LSP literal that uses `Plugin:` field
- Any other tests referencing `LSP.Plugin`

## Why this is Phase 1
All subsequent Codex beans depend on the Plugin field being generic. This must land first.

## Implementation Plan

### Approach

Pure data-structure refactoring: replace `LSP.Plugin string` with `LSP.Plugins map[string]string`, update the single template that consumes it, and fix all tests. No behavioral changes; the Claude settings output remains byte-identical before and after this change (aside from fixing a latent comma bug in the template -- see Step 3). The `claudePlugin` FuncMap helper keeps the template logic simple and isolates the "which tool are we extracting for" decision to a single Go function.

Only `CodingToolClaude` is added as a constant. `CodingToolCodex` is intentionally deferred until the Codex support bean that actually uses it, to avoid dead code that lint and review will flag.

### Files to Create/Modify

1. **`internal/stack/stack.go`** -- Replace `Plugin string` field with `Plugins map[string]string` in the `LSP` struct, add `CodingToolClaude` constant, update all 5 registry entries, update `copyStack()` to clone the map.
2. **`internal/render/claude.go`** -- Add `claudePlugin` FuncMap helper to `claudeFuncMap`.
3. **`internal/render/templates/claude-user-settings.json.tmpl`** -- Replace `.Plugin` access with the new `claudePlugin` pipe; fix latent comma logic bug.
4. **`internal/stack/stack_test.go`** -- Update tests that reference `LSP.Plugin` to use `LSP.Plugins`.
5. **`internal/render/claude_test.go`** -- Update isolation tests that construct `stack.LSP` literals with `Plugin:` to use `Plugins:`. Update structural assertions from `.Plugin` to `Plugins[stack.CodingToolClaude]`.
6. **`internal/render/render_test.go`** -- Update `TestMerge_MultipleStacks` which spot-checks `l.Plugin` to use `l.Plugins[stack.CodingToolClaude]`.
7. **`internal/render/dockerfile_test.go`** -- Update `TestDockerfile_DirectConfig_CustomRuntimesAndLSPs` (line 469) which constructs `stack.LSP{..., Plugin: "zls"}` to use `Plugins: map[string]string{stack.CodingToolClaude: "zls"}`.

### Steps

#### 1. Update `internal/stack/stack.go` -- Data structure change

- Add `"maps"` to the import list.
- Add one exported constant below the `StackID` constants block:
  ```go
  // CodingToolClaude identifies Claude Code in the LSP Plugins map.
  const CodingToolClaude = "claude"
  ```
  Note: `CodingToolCodex` is intentionally omitted. It will be added by the Codex support bean that first uses it, keeping this codebase free of dead code.
- In the `LSP` struct (lines 29-33), replace the `Plugin string` field with `Plugins map[string]string`. Update the doc comment to describe it as a map keyed by coding tool identifier (e.g., `CodingToolClaude`), with values being the tool-specific plugin identifier. Note that the map may be empty (e.g., Ruby has no official plugins for any tool).
- Update each of the 5 registry entries:
  - **Go** (line 88): `Plugin: "gopls-lsp@claude-plugins-official"` becomes `Plugins: map[string]string{CodingToolClaude: "gopls-lsp@claude-plugins-official"}`
  - **Node** (line 105): `Plugin: "typescript-lsp@claude-plugins-official"` becomes `Plugins: map[string]string{CodingToolClaude: "typescript-lsp@claude-plugins-official"}`
  - **Python** (line 123): `Plugin: "pyright-lsp@claude-plugins-official"` becomes `Plugins: map[string]string{CodingToolClaude: "pyright-lsp@claude-plugins-official"}`
  - **Rust** (line 140): `Plugin: "rust-analyzer-lsp@claude-plugins-official"` becomes `Plugins: map[string]string{CodingToolClaude: "rust-analyzer-lsp@claude-plugins-official"}`
  - **Ruby** (line 158): `Plugin: ""` becomes `Plugins: map[string]string{}`
- In `copyStack()` (lines 208-216), add `cp.LSP.Plugins = maps.Clone(s.LSP.Plugins)` to deep-copy the map. Place it after the existing slice clones. The existing slice clones remain unchanged.

#### 2. Update `internal/render/claude.go` -- FuncMap helper

- Add `"github.com/bjro/agentbox/internal/stack"` to the import list.
- Add a `"claudePlugin"` entry to `claudeFuncMap` (lines 11-24). The helper function signature is `func(plugins map[string]string) string` and it returns `plugins[stack.CodingToolClaude]` (zero value `""` if key absent).
- The `claudeTemplates` variable (lines 30-32) does not need changes since the template file names remain the same.

#### 3. Update `internal/render/templates/claude-user-settings.json.tmpl` -- Template syntax + comma fix

Current line 6:
```
  "enabledPlugins": { {{- range $i, $lsp := .LSPs}}{{if $lsp.Plugin}}{{if $i}}, {{end}}"{{$lsp.Plugin | jsonString}}": true{{end}}{{end}} }
```

The pre-existing comma logic has a latent bug: `{{if $i}}` uses the range index, not a count of emitted plugins. If the first LSP in sort order has no claude plugin and a later LSP does, the later LSP emits a leading comma because its index is non-zero, producing invalid JSON like `{ , "plugin": true }`. This does not trigger today because `solargraph` (Ruby, the only pluginless LSP) sorts after all plugin-bearing packages, but it will break as soon as a pluginless LSP with a lower-sorting package name is added.

Replace with a separator-variable approach that fixes this edge case:
```
  "enabledPlugins": { {{- $sep := ""}}{{range $lsp := .LSPs}}{{$p := .Plugins | claudePlugin}}{{if $p}}{{$sep}}"{{$p | jsonString}}": true{{$sep = ", "}}{{end}}{{end}} }
```

The separator starts as `""` and becomes `", "` after the first plugin is emitted. This avoids adding a new FuncMap helper and produces correct JSON regardless of which LSPs have claude plugins.

#### 4. Update `internal/stack/stack_test.go` -- Stack registry tests

- **`TestGet_ExistingStack`** (lines 24-60): Replace the comment on line 51 (`// Plugin may be empty`) with a structural assertion: `if s.LSP.Plugins == nil { t.Error("LSP.Plugins is nil, want non-nil (possibly empty) map") }`.
- **Add `TestPlugins_NonNil`**: Iterate all stacks via `All()`, assert `s.LSP.Plugins != nil` for each. This mirrors the pattern of `TestSystemDeps_NonNil` and `TestDevTools_NonNil`.
- **Add `TestPlugins_DefensiveCopy`**: Get a stack with plugins (e.g., Go), mutate the returned `Plugins` map by adding `"evil": "evil-plugin"`, get the same stack again, assert the mutation did not leak. Mirrors `TestSystemDeps_DefensiveCopy`.
- **Add `TestPlugins_KnownValues`**: Spot-check that `Get(Go).LSP.Plugins[CodingToolClaude]` equals `"gopls-lsp@claude-plugins-official"`. Spot-check that `Get(Ruby).LSP.Plugins` is empty (len 0). Mirrors `TestSystemDeps_KnownValues`.

#### 5. Update `internal/render/claude_test.go` -- Template/render tests

- **`TestRenderClaude_UserSettings_PluginsMatchRegistry`** (lines 91-130): Change `lsp.Plugin` references to `lsp.Plugins[stack.CodingToolClaude]` on lines 110, 113-114, 121-122. The import of `"github.com/bjro/agentbox/internal/stack"` is already present.
- **`TestRenderClaude_UserSettings_PluginsSpotCheck`** (lines 132-154): No changes needed -- it checks the rendered JSON keys, not the Go struct field.
- **`TestRenderClaude_UserSettings_NoDuplicatePlugins`** (lines 178-206): Change `lsp.Plugin` references to `lsp.Plugins[stack.CodingToolClaude]` on line 199.
- **`TestRenderClaude_DirectConfig_EmptyLSPs`** (lines 304-328): No changes -- constructs `GenerationConfig` with empty `[]stack.LSP{}`.
- **`TestRenderClaude_DirectConfig_CustomPlugins`** (lines 330-358): Change `Plugin: "custom-plugin"` to `Plugins: map[string]string{stack.CodingToolClaude: "custom-plugin"}` and same for `"another-plugin"`.
- **`TestRenderClaude_DirectConfig_PluginWithSpecialChars`** (lines 360-387): Change `Plugin: "quote\"and\\backslash"` to `Plugins: map[string]string{stack.CodingToolClaude: "quote\"and\\backslash"}`.
- **Add `TestRenderClaude_DirectConfig_NonClaudePluginsIgnored`**: Construct a `GenerationConfig` with one LSP whose `Plugins` map has only a `"codex"` key (no `"claude"` key). Render and verify `enabledPlugins` is empty `{}`. This proves the `claudePlugin` helper correctly filters to claude-only plugins.
- **Add `TestRenderClaude_DirectConfig_MixedPlugins`**: Construct a `GenerationConfig` with one LSP whose `Plugins` map has both `"claude"` and `"codex"` keys. Render and verify only the claude plugin appears in `enabledPlugins`.
- **Add `TestRenderClaude_DirectConfig_PluginlessFirstLSP`**: Construct a `GenerationConfig` with two LSPs where the first (sorted by Package) has no claude plugin and the second does. Render, unmarshal as JSON, and verify valid JSON with correct enabledPlugins. This is the regression test for the comma logic fix in Step 3. The first LSP should have a Package name that sorts before the plugin-bearing one (e.g., `"aaa-lsp"` with `Plugins: map[string]string{}` and `"zzz-lsp"` with `Plugins: map[string]string{stack.CodingToolClaude: "zzz-plugin"}`).

#### 6. Update `internal/render/render_test.go` -- Merge tests

- **`TestMerge_MultipleStacks`** (lines 70-80): Change `plugins[l.Plugin] = true` to `plugins[l.Plugins[stack.CodingToolClaude]] = true`. The import of `"github.com/bjro/agentbox/internal/stack"` is already present.

#### 7. Update `internal/render/dockerfile_test.go` -- Dockerfile isolation test

- **`TestDockerfile_DirectConfig_CustomRuntimesAndLSPs`** (line 469): Change `Plugin: "zls"` to `Plugins: map[string]string{stack.CodingToolClaude: "zls"}`. This test constructs a hand-built `GenerationConfig` to test Dockerfile rendering and does not exercise the Plugin field in any assertion (it only checks `InstallCmd` and system deps), so the value is arbitrary but must compile. The file already imports `"github.com/bjro/agentbox/internal/stack"` for `stack.StackID`.

### Testing Strategy

**TDD sequence** (tests written/updated before implementation changes):

1. **Update test files first** so they compile-fail against the old `Plugin` field, confirming the test surface covers the change.
2. **Then update production code** to make tests pass.

**Tests to write (new):**
- `TestPlugins_NonNil` -- structural invariant: all stacks have non-nil Plugins map
- `TestPlugins_DefensiveCopy` -- mutation guard: map clone prevents registry corruption
- `TestPlugins_KnownValues` -- spot-check: Go has claude plugin, Ruby has empty map
- `TestRenderClaude_DirectConfig_NonClaudePluginsIgnored` -- codex-only LSP produces empty enabledPlugins
- `TestRenderClaude_DirectConfig_MixedPlugins` -- mixed map only emits claude plugin
- `TestRenderClaude_DirectConfig_PluginlessFirstLSP` -- regression test for comma logic fix

**Tests to update (existing):**
- `TestGet_ExistingStack` -- add Plugins nil check
- `TestRenderClaude_UserSettings_PluginsMatchRegistry` -- `.Plugin` to `.Plugins[CodingToolClaude]`
- `TestRenderClaude_UserSettings_NoDuplicatePlugins` -- `.Plugin` to `.Plugins[CodingToolClaude]`
- `TestRenderClaude_DirectConfig_CustomPlugins` -- `Plugin:` to `Plugins:`
- `TestRenderClaude_DirectConfig_PluginWithSpecialChars` -- `Plugin:` to `Plugins:`
- `TestMerge_MultipleStacks` -- `.Plugin` to `.Plugins[CodingToolClaude]`
- `TestDockerfile_DirectConfig_CustomRuntimesAndLSPs` -- `Plugin:` to `Plugins:`

**Verification commands:**
- `go build ./...` -- confirms compilation
- `go test ./internal/stack/...` -- stack registry tests
- `go test ./internal/render/...` -- render and template tests
- `go test ./...` -- all unit tests
- `go test -tags integration ./...` -- including integration tests
- `golangci-lint run ./...` -- lint clean

### No ADR Needed

This is a data-structure field rename with a FuncMap helper addition. It does not introduce new dependencies, architectural patterns, or cross-cutting concerns. The existing registry pattern (ADR-0004) and template rendering conventions (ADR-0006) remain unchanged. The `maps.Clone` usage for defensive copying follows the existing `slices.Clone` pattern already established in `copyStack()`.

### Open Questions

None. The scope is well-defined: 3 production files, 1 template file, and 4 test files. The behavior is identical before and after the change (aside from the comma logic fix, which only affects an edge case that does not trigger with the current registry data).

## Definition of Done

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [ ] Branch pushed to remote
- [ ] PR created
- [x] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review
