---
# agentbox-comb
title: Update integration tests for Codex support
status: todo
type: task
priority: normal
created_at: 2026-04-08T09:16:54Z
updated_at: 2026-04-08T09:17:47Z
parent: agentbox-cqi5
blocked_by:
    - agentbox-qr1p
    - agentbox-vr4z
    - agentbox-0w8k
    - agentbox-6o49
    - agentbox-thmk
---

Update CLI integration tests to cover Codex files in the init and update pipelines.

## Scope

### `cmd/init_integration_test.go`
- Add `codex-config.toml` and `sync-codex-settings.sh` to `expectedFiles` slice
- Assert both files exist and are non-empty after `agentbox init`
- Assert `sync-codex-settings.sh` has executable permissions (mode & 0o111)
- Assert `codex-config.toml` content contains expected TOML keys
- Update file count / manifest assertions

### `cmd/update_integration_test.go`
- Assert Codex files are regenerated on `agentbox update`
- Assert Codex files survive the update cycle (not deleted)
- Update `seedInitDir` helper if needed to produce Codex files

### Note
This bean covers only integration test changes. Unit tests for individual render functions are covered in their respective beans (codex settings, dockerfile, devcontainer, firewall).

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
