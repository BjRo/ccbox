---
# agentbox-comb
title: Update integration tests for Codex support
status: in-progress
type: task
priority: normal
created_at: 2026-04-08T09:16:54Z
updated_at: 2026-04-08T14:25:56Z
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

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [x] Branch pushed to remote
- [x] PR created
- [x] Automated code review passed via `@review-backend` subagent (via Task tool)
- [x] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [x] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [x] All other checklist items above are completed
- [x] User notified for human review

## Implementation Plan

### Approach

Most of the init integration test work described in the original scope has already been completed as part of the blocking beans (primarily commit `529be90` which wired Codex into init/update commands and updated test expectations). The remaining work is in the **update integration tests**, which lack codex-specific content assertions, permission checks, and regeneration verification. There is also a small gap in the update unit tests.

The plan adds targeted assertions to existing update test functions and one new integration test that verifies codex files are freshly regenerated (not stale) after an update with a stack change.

### Current State Assessment

**Init integration tests (COMPLETE -- no changes needed)**:
- `expectedFiles` already includes `codex-config.toml` and `sync-codex-settings.sh` (lines 48-49)
- `executableScripts` already includes `sync-codex-settings.sh` (line 60)
- `TestIntegration_SingleGoStack` already asserts codex-config.toml contains `approval_policy`, `sandbox_mode`, and `apps = false` (lines 188-196)
- `TestIntegration_SingleGoStack` already asserts sync-codex-settings.sh references `codex-config.toml` and `$HOME/.codex` (lines 199-205)

**Update integration tests (GAPS)**: 
1. `TestIntegration_UpdatePreservesCustomizations` -- no codex content assertions after update
2. `TestIntegration_UpdateWithStackChange` -- no assertion that codex files exist or have correct content after stack change
3. `TestIntegration_UpdateForceMode` -- no assertion that codex files are regenerated during force mode
4. `TestIntegration_UpdateIdempotent` -- implicitly covers codex via `expectedFiles` loop (adequate)
5. No test checks executable permissions on scripts after `update` (not just codex -- this is a broader gap)

**Update unit tests (GAPS)**:
1. `TestUpdateCommand_ForceRegeneration` -- no assertion that codex files are produced
2. `TestUpdateCommand_RegeneratesAgentboxStage` -- only checks Dockerfile, does not verify other regenerated files

### Files to Modify

- `cmd/update_integration_test.go` -- Add codex content and permission assertions to existing tests; add one focused test for codex file regeneration after update
- `cmd/update_test.go` -- Add codex assertions to `TestUpdateCommand_ForceRegeneration`

### Steps

1. **Add codex content assertions to `TestIntegration_UpdatePreservesCustomizations`** -- After the existing Dockerfile and mise-config preservation assertions (line 70), add assertions that:
   - `codex-config.toml` exists and contains `approval_policy` (verifies regeneration)
   - `sync-codex-settings.sh` exists and contains `codex-config.toml` (verifies regeneration)
   - Both files are non-empty

2. **Add codex and permission assertions to `TestIntegration_UpdateWithStackChange`** -- After the existing Dockerfile and config assertions (line 112), add assertions that:
   - `codex-config.toml` exists, is non-empty, and contains expected TOML keys (`approval_policy`, `sandbox_mode`, `apps = false`)
   - `sync-codex-settings.sh` exists, is non-empty, and references `codex-config.toml` and `$HOME/.codex`
   - All `executableScripts` have executable permissions after the update (covers `sync-codex-settings.sh` plus the others)

3. **Add permission assertions to `TestIntegration_UpdateForceMode`** -- After the existing custom stage assertions (line 162), add assertions that:
   - All `executableScripts` have executable permissions after `--force` update
   - `codex-config.toml` exists and is non-empty (force mode regenerates everything)

4. **Add codex assertion to unit test `TestUpdateCommand_ForceRegeneration`** in `cmd/update_test.go` -- After the existing `FROM agentbox AS custom` assertion (line 127), add assertions that:
   - `codex-config.toml` exists and is non-empty in the output directory
   - `sync-codex-settings.sh` exists and is non-empty in the output directory

5. **Verify all tests pass** -- Run `go test ./...` and `go test -tags integration ./...` and `golangci-lint run ./...`

### Testing Strategy

- All changes are test code; no production code changes
- Run `go test -tags integration ./cmd/...` to verify all integration tests pass (including the new assertions)
- Run `go test ./cmd/...` to verify unit tests pass
- Run `golangci-lint run ./...` to verify no lint issues
- Verify the new assertions fail when codex files are absent (manually confirm by temporarily commenting out codex rendering in `renderFiles` -- but do not commit this)

### Scope Boundaries

- No changes to `cmd/init_integration_test.go` -- all init-side codex assertions are already present
- No changes to production code (`cmd/init.go`, `cmd/update.go`, `internal/render/codex.go`)
- No new test functions needed in the integration test file -- all additions fit naturally into existing test functions
- The `seedInitDir` helper does not need changes -- it already produces codex files via `init --stack`
- No ADR needed -- this is purely additive test coverage with no architectural changes

### Open Questions

- The blocker `agentbox-thmk` (Update generated README to document Codex CLI) is still in `todo` status. This bean should not be blocked by it since README content changes do not affect the codex file integration test assertions described here. However, if README template changes introduce new content that should be spot-checked in integration tests, that would be a separate concern.

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-08T14:00:00Z |
| challenge | completed | 1 | 2026-04-08 |
| implement | completed | 1 | 2026-04-08 |
| pr | completed | 1 | 2026-04-08 |
| review | completed | 2 | 2026-04-08 |
| codify | completed | 1 | 2026-04-08 |
