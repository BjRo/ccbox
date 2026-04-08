---
# agentbox-kyij
title: Refactor LSP Plugin field to be tool-agnostic
status: todo
type: task
created_at: 2026-04-08T09:15:46Z
updated_at: 2026-04-08T09:15:46Z
parent: agentbox-cqi5
---

Refactor `LSP.Plugin string` in `internal/stack/stack.go` to `LSP.Plugins map[string]string`, keyed by coding tool identifier. This decouples the stack registry from any single coding tool and prepares for Codex (and future tools) that may have their own plugin systems.

## Scope

### `internal/stack/stack.go`
- Replace `Plugin string` in `LSP` struct with `Plugins map[string]string`
- Define constants: `CodingToolClaude = "claude"`, `CodingToolCodex = "codex"`
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
- No structural changes needed — `LSP` struct change flows through `GenerationConfig.LSPs`

### Tests to update
- `internal/stack/stack_test.go` — update for Plugins map
- `internal/render/claude_test.go` — update for Plugins map, test new FuncMap helper
- Any other tests referencing `LSP.Plugin`

## Why this is Phase 1
All subsequent Codex beans depend on the Plugin field being generic. This must land first.

## Definition of Done

- [ ] Tests written (TDD: write tests before implementation)
- [ ] No new TODO/FIXME/HACK/XXX comments introduced
- [ ] `golangci-lint run ./...` passes with no errors
- [ ] `go test ./...` passes with no failures
- [ ] Branch pushed to remote
- [ ] PR created
- [ ] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review
