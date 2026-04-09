---
# agentbox-31g0
title: Update root README.md for Codex CLI support
status: todo
type: task
priority: normal
created_at: 2026-04-09T09:03:43Z
updated_at: 2026-04-09T09:03:50Z
---

The root README.md is outdated after the Codex CLI epic. Needs updates:

1. Intro and features list — mention Codex CLI alongside Claude Code
2. Dockerfile section — add @openai/codex and bubblewrap
3. Firewall always-on table — add all OpenAI domains (api.openai.com, auth.openai.com, auth0.openai.com, chatgpt.com, accounts.openai.com)
4. Settings sync section — document Codex copy-on-first-run strategy
5. devcontainer.json section — add containerEnv (OPENAI_API_KEY), Codex volume mount, Codex extension
6. Generated files table — add codex-config.toml, sync-codex-settings.sh, update config.toml to mise-config.toml
7. Go version — fix 1.25+ to 1.24+

## Definition of Done

- [x] Tests written (TDD: write tests before implementation) -- N/A: documentation-only change, no testable behavior
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [x] Branch pushed to remote
- [ ] PR created
- [ ] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review
