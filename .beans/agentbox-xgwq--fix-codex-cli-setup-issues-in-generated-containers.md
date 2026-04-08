---
# agentbox-xgwq
title: Fix Codex CLI setup issues in generated containers
status: in-progress
type: bug
priority: normal
created_at: 2026-04-08T12:14:48Z
updated_at: 2026-04-08T12:29:06Z
parent: agentbox-cqi5
---

Two issues found when running Codex CLI in a generated devcontainer:

1. **Missing bubblewrap**: Warning 'could not find system bubblewrap on PATH'. Codex uses it for sandboxing. Since we set sandbox_mode=danger-full-access the container is the sandbox, but the warning is noisy. Fix: add `bubblewrap` to apt-get install in Dockerfile.tmpl.

2. **codex_apps MCP timeout**: Warning 'MCP client for codex_apps timed out after 30 seconds'. The built-in MCP server for web apps fails to start — likely needs additional allowlisted domains or should be disabled in the generated config. Fix: either add `startup_timeout_sec` to codex-config.toml.tmpl, disable the MCP server, or allowlist the required domains.

Also update the project's own .devcontainer to match.

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

Two targeted template fixes plus a project .devcontainer refresh. Both issues are straightforward:

1. **bubblewrap**: Add `bubblewrap` to the static system packages list in `Dockerfile.tmpl`. This is the `bwrap` binary that Codex CLI probes on startup for its Linux sandbox. Even though `sandbox_mode = "danger-full-access"` means Codex does not actually sandbox commands, the CLI still checks for `bwrap` and emits a noisy warning if absent. Adding it as a system package silences the warning at negligible image size cost (~45 KB).

2. **codex_apps MCP timeout**: Disable the `apps` feature in `codex-config.toml.tmpl` via `[features]` / `apps = false`. The `apps` feature (previously called `codex_apps`) is a built-in MCP server that Codex CLI starts for interactive web app previews. Inside network-isolated containers it cannot reach external services and times out after 30 seconds, producing a noisy warning on every startup. Disabling it is the correct fix because:
   - The alternative of allowlisting domains is infeasible (the app server uses arbitrary external URLs).
   - Increasing the timeout would just delay the failure.
   - Web app previews are not useful inside a headless devcontainer.

### Files to Create/Modify

#### Templates (source of truth for generated output)

- `internal/render/templates/Dockerfile.tmpl` (line 17) -- Add `bubblewrap` to the system packages `apt-get install` block.
- `internal/render/templates/codex-config.toml.tmpl` -- Add `[features]` section with `apps = false`.

#### Tests

- `internal/render/dockerfile_test.go` -- Add test asserting `bubblewrap` is present in rendered Dockerfile output.
- `internal/render/codex_test.go` -- Update existing tests to assert `[features]` section and `apps = false` are present in rendered config. Update the `IsStatic` test since the template is still static.

#### Project's own .devcontainer (dogfooding)

- `.devcontainer/Dockerfile` -- Add `bubblewrap` to the apt-get install line (managed stage, line 19).
- `.devcontainer/codex-config.toml` -- Add `[features]` section with `apps = false`.

### Steps

1. **Add bubblewrap to Dockerfile template**
   - File: `internal/render/templates/Dockerfile.tmpl`
   - In the `# --- System packages ---` RUN block (line 17), add `bubblewrap` after `build-essential` in the static package list, before the `jq fzf` packages. The exact insertion point should maintain alphabetical ordering within the "core tooling" section: `build-essential bubblewrap jq fzf`.
   - The `{{ range .SystemDeps }}` block follows immediately after, so no template logic changes needed.

2. **Add `[features]` section to codex-config.toml template**
   - File: `internal/render/templates/codex-config.toml.tmpl`
   - Append a new `[features]` TOML section after the existing `sandbox_mode` line:
     ```
     [features]
     apps = false
     ```
   - This remains a static template (no Go template actions), which is the established pattern per `TestRenderCodex_Config_IsStatic`.

3. **Write/update tests (TDD -- these should be written first)**

   a. **Dockerfile bubblewrap test** in `internal/render/dockerfile_test.go`:
      - Add `"bubblewrap"` to the `required` slice in `TestDockerfile_AlwaysIncludedPackages` (line 44). This test already asserts that always-present apt packages appear in every rendered Dockerfile. Adding bubblewrap here is the natural fit -- it is a system package that should be present regardless of which stacks are selected.

   b. **Codex config `[features]` test** in `internal/render/codex_test.go`:
      - Add a new test `TestRenderCodex_Config_AppsDisabled` that renders the config template and asserts both `[features]` and `apps = false` appear in the output.
      - Update `TestRenderCodex_Config_SpecificValues` to also assert `apps = false` is present among the expected values.
      - The `IsStatic` tests (`TestRenderCodex_Config_IsStatic`) need no changes since the template remains static.

4. **Update project's own .devcontainer/Dockerfile**
   - File: `.devcontainer/Dockerfile`
   - Add `bubblewrap` to the apt-get install line in the managed stage (line 19). Insert between `build-essential` and `jq` to match the template.

5. **Update project's own .devcontainer/codex-config.toml**
   - File: `.devcontainer/codex-config.toml`
   - Add:
     ```
     [features]
     apps = false
     ```
     after the existing `sandbox_mode` line.

### Testing Strategy

- **Unit tests** (`go test ./internal/render/...`):
  - `TestDockerfile_AlwaysIncludedPackages`: Verify `bubblewrap` is in the always-present system packages list.
  - `TestRenderCodex_Config_AppsDisabled`: New test verifying `[features]` and `apps = false` appear in rendered codex config.
  - `TestRenderCodex_Config_SpecificValues`: Updated to include `apps = false`.
  - All existing Dockerfile tests must continue to pass (shell syntax validation, no triple newlines, determinism, no template artifacts).
  - All existing Codex tests must continue to pass (static template verification, determinism, no template artifacts).

- **Integration tests** (`go test -tags integration ./cmd/...`):
  - `TestIntegration_SingleGoStack`: Already checks codex-config.toml for `approval_policy` and `sandbox_mode`. The existing assertions will still pass. No new assertions needed here unless we want to spot-check the new `apps = false` line (optional but lightweight to add).
  - All other integration tests should pass unchanged since neither fix changes the file manifest, executable script list, or structural devcontainer.json format.

- **Lint**: `golangci-lint run ./...` must pass.
- **Full suite**: `go test ./...` for unit tests, `go test -tags integration ./...` for integration.

### Open Questions

None. Both fixes are well-understood, minimal changes with clear rationale.

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-08 |
| challenge | in-progress | 1 | 2026-04-08 |
| implement | in-progress | 1 | 2026-04-08 |
| pr | in-progress | 1 | 2026-04-08 |
| review | completed | 2 | 2026-04-08 |
| codify | completed | 1 | 2026-04-08 |
