---
# agentbox-thmk
title: Update generated README to document Codex CLI
status: todo
type: task
priority: normal
created_at: 2026-04-08T09:16:43Z
updated_at: 2026-04-08T09:17:47Z
parent: agentbox-cqi5
blocked_by:
    - agentbox-0w8k
---

Update the generated README template to document Codex CLI availability alongside Claude Code, including auth options.

## Scope

### `internal/render/templates/README.md.tmpl`
- Document both Claude Code and Codex CLI are available in the container
- Document Codex auth options:
  - **OPENAI_API_KEY**: Set on host before building container, automatically forwarded via `containerEnv`
  - **ChatGPT login**: Run `codex` and select "Sign in with ChatGPT" — login persists across container rebuilds via volume mount
- Document Codex usage: `codex` for interactive TUI, `codex --full-auto` for autonomous mode

### Tests
- Update README render tests to assert Codex documentation appears

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
