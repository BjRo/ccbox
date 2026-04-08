---
# agentbox-sukl
title: Add Codex to agentbox's own .devcontainer (dogfooding)
status: todo
type: task
created_at: 2026-04-08T09:17:05Z
updated_at: 2026-04-08T09:17:05Z
parent: agentbox-cqi5
---

Update agentbox's own .devcontainer setup to include Codex CLI, dogfooding the same pattern we generate for users.

## Scope

### `.devcontainer/Dockerfile` (custom stage)
- Add `RUN npm install -g @openai/codex` in the custom stage

### `.devcontainer/devcontainer.json`
- Add `"openai.chatgpt"` to VS Code extensions array
- Add `"source=agentbox-codex-config,target=/home/node/.codex,type=volume"` to mounts
- Add `containerEnv` with `"OPENAI_API_KEY": "${localEnv:OPENAI_API_KEY}"`
- Update `postStartCommand`:
  - Add `/home/node/.codex` to the chown command
  - Add Codex settings sync step (or create a manual codex-config.toml and sync script in our own .devcontainer)

### Codex settings for our own container
- Create `.devcontainer/codex-config.toml` with appropriate settings (full-auto + danger-full-access sandbox, since the container is externally sandboxed)
- Create `.devcontainer/sync-codex-settings.sh` or reuse the generated pattern

### Note
This is independent of all other beans — can be done in parallel. No Go code changes, just devcontainer config files.

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
