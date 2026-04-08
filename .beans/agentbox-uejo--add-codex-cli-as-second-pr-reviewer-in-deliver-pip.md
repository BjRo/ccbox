---
# agentbox-uejo
title: Add Codex CLI as second PR reviewer in deliver pipeline
status: todo
type: feature
priority: normal
created_at: 2026-04-08T12:45:09Z
updated_at: 2026-04-08T12:45:15Z
parent: agentbox-cqi5
---

Add a @review-codex subagent that runs `codex exec review --base main` and posts findings as PR comments. Integrate into the deliver pipeline alongside @review-backend for parallel independent reviews.

## Scope

### New files
- `.claude/agents/review-codex.md` — Subagent that runs `codex exec review --base main --full-auto --sandbox read-only -o /tmp/codex-review.md`, reads the output, and posts findings as a PR comment via `gh pr comment`.

### Modified files
- `.claude/skills/deliver/SKILL.md` — Update Step 3 to launch both @review-backend and @review-codex in parallel. Step 4 evaluates findings from both; any findings from either trigger rework.

### Design decisions
- **review-codex as a subagent** (not a skill): Same interface as review-backend — posts findings as PR comments. Deliver pipeline treats both reviewers uniformly.
- **Parallel execution**: Both reviews launch simultaneously to minimize wall-clock time.
- **Rework picks up both**: The rework agent already reads all PR comments, so it naturally addresses findings from both reviewers.
- **Re-review loop**: After rework, both reviewers re-run. Both must be clean to proceed.
- **Model**: Use highest-quality available (configurable in agent).
- **Sandbox**: read-only (reviews don't need writes).
- **Output capture**: `-o /tmp/codex-review.md` for the agent to read and post.

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
