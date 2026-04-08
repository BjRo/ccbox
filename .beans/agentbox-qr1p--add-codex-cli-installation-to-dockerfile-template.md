---
# agentbox-qr1p
title: Add Codex CLI installation to Dockerfile template
status: todo
type: task
priority: normal
created_at: 2026-04-08T09:15:54Z
updated_at: 2026-04-08T09:17:47Z
parent: agentbox-cqi5
blocked_by:
    - agentbox-kyij
---

Add `npm install -g @openai/codex` to the generated Dockerfile, alongside the existing Claude Code install.

## Scope

### `internal/render/templates/Dockerfile.tmpl`
- After line 69 (`RUN npm install -g @anthropic-ai/claude-code`), add:
  ```
  # --- Codex CLI ---
  RUN npm install -g @openai/codex
  ```

### `internal/render/templates/custom-stage.tmpl`
- Update the comment block to mention Codex alongside Claude Code:
  `# The agentbox stage above provides: mise runtimes, LSP servers, Claude Code, Codex CLI,`

### Tests
- `internal/render/dockerfile_test.go`:
  - Assert `@openai/codex` appears in rendered output
  - Update ordering tests (Codex after Claude Code, before USER root)
  - Existing Claude Code install assertions remain unchanged

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
