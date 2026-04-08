---
# agentbox-0w8k
title: Add Codex to devcontainer.json template (extension, volume, env, postStart)
status: todo
type: task
priority: normal
created_at: 2026-04-08T09:16:37Z
updated_at: 2026-04-08T09:17:47Z
parent: agentbox-cqi5
blocked_by:
    - agentbox-vr4z
---

Update the devcontainer.json template to include Codex VS Code extension, persistent volume mount, environment variable pass-through, and settings sync in the startup command.

## Scope

### `internal/render/templates/devcontainer.json.tmpl`

1. **VS Code extension**: Add `"openai.chatgpt"` to the extensions array (alongside `"anthropic.claude-code"`)

2. **Volume mount**: Add `"source=agentbox-codex-config,target=/home/node/.codex,type=volume"` to the mounts array for Codex state persistence (auth tokens, config, memories)

3. **containerEnv** (new field): Add environment variable pass-through for API key auth:
   ```json
   "containerEnv": {
     "OPENAI_API_KEY": "${localEnv:OPENAI_API_KEY}"
   }
   ```
   This passes the host env var if set; if not set, Codex falls back to ChatGPT interactive login (tokens persisted via volume mount).

4. **postStartCommand**: Update to include Codex setup:
   - Add `/home/node/.codex` to the `chown` command
   - Add `bash .devcontainer/sync-codex-settings.sh` to the command chain (after sync-claude-settings.sh, before init-firewall.sh)

### Tests
- `internal/render/devcontainer_test.go`:
  - Assert `openai.chatgpt` in extensions array
  - Assert `agentbox-codex-config` volume mount present
  - Assert `OPENAI_API_KEY` in containerEnv
  - Assert `sync-codex-settings.sh` in postStartCommand
  - Assert `/home/node/.codex` in chown command
  - JSON unmarshal validity still passes

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
