---
# agentbox-96ja
title: Rename config.toml to mise-config.toml
status: in-progress
type: task
priority: normal
created_at: 2026-04-08T12:04:48Z
updated_at: 2026-04-08T12:36:18Z
parent: agentbox-cqi5
---

Now that codex-config.toml exists in .devcontainer/, the generic config.toml name is ambiguous. Rename it to mise-config.toml for clarity. Affects: Dockerfile.tmpl COPY directive, config.toml.tmpl filename, cmd/init.go and cmd/update.go file maps, the project's own .devcontainer/config.toml, and update command's config preservation logic.

## Implementation Plan

### Approach

Rename every reference to the mise `config.toml` file to `mise-config.toml` throughout the codebase. The template file (`mise-config.toml.tmpl`) was already renamed in a prior commit, so the template itself does not need to change. The rename touches: the Dockerfile template's COPY directive, the README template's user-facing docs, the `cmd/init.go` and `cmd/update.go` file maps, all test files that reference the old name, the project's own `.devcontainer/config.toml` file, and documentation (ADRs, rules).

Key distinction: references to `~/.codex/config.toml` (Codex CLI's own config path) and generic example references in `custom-stage.tmpl` must NOT be renamed -- they are unrelated to the mise config.

### Files to Create/Modify

#### Templates
- `internal/render/templates/Dockerfile.tmpl` -- Rename `COPY config.toml` to `COPY mise-config.toml` and update the accompanying comment
- `internal/render/templates/README.md.tmpl` -- Rename all 3 occurrences of `.devcontainer/config.toml` to `.devcontainer/mise-config.toml`

#### Production Code
- `cmd/init.go` -- Change file map key from `"config.toml"` to `"mise-config.toml"` in `renderFiles`
- `cmd/update.go` -- Rename `configTomlPath` variable to use `mise-config.toml`, update file map key, update Long description and comments

#### Unit Tests
- `cmd/init_test.go` -- Update `expectedInitFiles` slice, update all path references from `config.toml` to `mise-config.toml`, update file map key lookups
- `cmd/update_test.go` -- Update `configPath` references in `TestUpdateCommand_PreservesConfigToml`
- `internal/render/dockerfile_test.go` -- Update all COPY directive assertion strings (approximately 7 occurrences)
- `internal/render/readme_test.go` -- Update assertion string for config.toml mention

#### Integration Tests
- `cmd/init_integration_test.go` -- Update `expectedFiles` slice, update all content assertion paths and strings (approximately 6 occurrences)
- `cmd/update_integration_test.go` -- Update `configPath` references

#### Project's Own Devcontainer
- `.devcontainer/config.toml` -- Rename the file itself to `.devcontainer/mise-config.toml`
- `.devcontainer/Dockerfile` -- Update COPY directive and comment
- `.devcontainer/README.md` -- Update all references

#### Documentation
- `decisions/0008-ensure-node-as-post-merge-invariant.md` -- Update `config.toml` references to `mise-config.toml`
- `decisions/0009-multi-stage-dockerfile-for-user-customizations.md` -- Update all `config.toml` references
- `decisions/README.md` -- Update table entry
- `.claude/rules/go-patterns.md` -- Update `config.toml` references in the Update Command section
- `.claude/rules/template-rendering.md` -- Update reference in Standalone Config Files section
- `.claude/rules/testing-patterns.md` -- Update `config.toml` references in the Update Command Test Pattern section

#### Files NOT to Modify (Codex CLI references)
- `internal/render/templates/sync-codex-settings.sh.tmpl` line 14 (`$SETTINGS_DIR/config.toml`) -- This is `~/.codex/config.toml`, the Codex CLI's own config path
- `internal/render/codex_test.go` line 109 -- Checking for `config.toml` in sync script output (Codex target)
- `.devcontainer/sync-codex-settings.sh` -- Output of sync-codex-settings template
- `internal/render/templates/custom-stage.tmpl` line 13 -- Generic example comment

### Steps

#### Step 1: Rename the project's own devcontainer file
- `git mv .devcontainer/config.toml .devcontainer/mise-config.toml`
- Update `.devcontainer/Dockerfile` lines 42-43: change comment to reference `mise-config.toml` and change `COPY config.toml` to `COPY mise-config.toml`
- Update `.devcontainer/README.md`: change all 3 occurrences of `.devcontainer/config.toml` to `.devcontainer/mise-config.toml`

#### Step 2: Update templates (TDD -- write test assertions first)

Update test files first:
- `internal/render/dockerfile_test.go` -- Change all assertion strings from `"COPY config.toml /home/node/.config/mise/config.toml"` to `"COPY mise-config.toml /home/node/.config/mise/config.toml"` (note: the destination path `/home/node/.config/mise/config.toml` stays the same because that is mise's expected config location)
- `internal/render/readme_test.go` -- Change assertion on line 262 from `config.toml` to `mise-config.toml`

Then update templates:
- `internal/render/templates/Dockerfile.tmpl` line 43: change comment to `# Managed by mise; edit .devcontainer/mise-config.toml to change versions.`
- `internal/render/templates/Dockerfile.tmpl` line 44: change to `COPY mise-config.toml /home/node/.config/mise/config.toml`
- `internal/render/templates/README.md.tmpl` lines 105, 122, 130: change `config.toml` to `mise-config.toml`

#### Step 3: Update production code (TDD -- write test assertions first)

Update test files first:
- `cmd/init_test.go`: Update `expectedInitFiles` (line 26), all `configPath` path constructions, and all `files["config.toml"]` lookups to use `"mise-config.toml"`
- `cmd/update_test.go`: Update `configPath` in `TestUpdateCommand_PreservesConfigToml` (line 170)
- `cmd/init_integration_test.go`: Update `expectedFiles` (line 51), all `configToml := readFile(...)` calls, and the COPY directive assertion strings
- `cmd/update_integration_test.go`: Update `configPath` (line 37)

Then update production code:
- `cmd/init.go` line 326: Change file map key from `"config.toml"` to `"mise-config.toml"`
- `cmd/update.go` line 28: Change `config.toml` to `mise-config.toml` in Long description
- `cmd/update.go` line 30: Change `.devcontainer/config.toml` to `.devcontainer/mise-config.toml` in Long description
- `cmd/update.go` line 98: Update comment to reference `mise-config.toml`
- `cmd/update.go` line 99: Change `"config.toml"` to `"mise-config.toml"` in path construction
- `cmd/update.go` line 124: Change `"config.toml"` to `"mise-config.toml"` in file map key

#### Step 4: Update documentation
- Update all references in `decisions/0008-ensure-node-as-post-merge-invariant.md`
- Update all references in `decisions/0009-multi-stage-dockerfile-for-user-customizations.md`
- Update the table entry in `decisions/README.md`
- Update references in `.claude/rules/go-patterns.md` (the "Update Command" section, lines 174-175)
- Update reference in `.claude/rules/template-rendering.md` (the "Standalone Config Files" section, line 116)
- Update references in `.claude/rules/testing-patterns.md` (lines 100, 102)

#### Step 5: Run tests and lint
- `go test ./...` -- verify all unit tests pass
- `go test -tags integration ./...` -- verify all integration tests pass
- `golangci-lint run ./...` -- verify lint passes

### Testing Strategy

This is a pure rename with no behavioral changes. The testing strategy is:

1. **Existing test coverage suffices**: All existing tests already verify the correct filenames, COPY directives, and file map keys. By updating the expected strings in tests FIRST (TDD), then updating the production code to match, we get immediate validation.

2. **Key assertions to verify after rename**:
   - `expectedInitFiles` / `expectedFiles` contain `"mise-config.toml"` (not `"config.toml"`)
   - Dockerfile render tests assert `"COPY mise-config.toml /home/node/.config/mise/config.toml"` (source renamed, destination unchanged)
   - README render test asserts output contains `"mise-config.toml"`
   - Update command preserves `mise-config.toml` (not `config.toml`)
   - Integration tests verify `mise-config.toml` exists with correct content

3. **No new tests needed**: The existing tests comprehensively cover the filename in file manifests, COPY directives, content assertions, and preservation logic. The rename is purely mechanical.

4. **Verify Codex references unchanged**: After renaming, confirm that `sync-codex-settings.sh` still references `$HOME/.codex/config.toml` (not renamed) and `codex_test.go` still passes.

### Open Questions

None. The scope is clear: rename mise's `config.toml` to `mise-config.toml` in all generated output, tests, documentation, and the project's own devcontainer. Leave Codex CLI's `~/.codex/config.toml` path and generic examples untouched.

## Definition of Done

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [x] Branch pushed to remote
- [x] PR created
- [x] Automated code review passed via `@review-backend` subagent (via Task tool)
- [x] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [x] ADR written via `/decision` skill — N/A, pure rename with no architectural changes
- [x] All other checklist items above are completed
- [ ] User notified for human review

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | done | 1 | 2026-04-08 |
| challenge | done | 1 | 2026-04-08 |
| implement | done | 1 | 2026-04-08 |
| pr | done | 1 | 2026-04-08 |
| review | done | 2 | 2026-04-08 |
| codify | done | 1 | 2026-04-08 |
