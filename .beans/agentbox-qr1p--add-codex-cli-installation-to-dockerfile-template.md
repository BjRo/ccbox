---
# agentbox-qr1p
title: Add Codex CLI installation to Dockerfile template
status: completed
type: task
priority: normal
created_at: 2026-04-08T09:15:54Z
updated_at: 2026-04-08T11:09:10Z
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

## Implementation Plan

### Approach

Add a static `RUN npm install -g @openai/codex` line to the Dockerfile template immediately after the existing Claude Code install. This is a static template change with no new Go code, no new template variables, and no `GenerationConfig` changes. The custom stage comment gets updated to mention Codex CLI.

### Files to Modify

- `internal/render/templates/Dockerfile.tmpl` -- Add a new `# --- Codex CLI ---` section with `RUN npm install -g @openai/codex` after the Claude Code install (line 69), before the `USER root` line (line 71).
- `internal/render/templates/custom-stage.tmpl` -- Change the comment on line 19 from "The agentbox stage above provides: mise runtimes, LSP servers, Claude Code," to "The agentbox stage above provides: mise runtimes, LSP servers, Claude Code, Codex CLI,".
- `internal/render/dockerfile_test.go` -- Add and update tests (see Testing Strategy below).

### Steps

1. **Add Codex install to Dockerfile template** -- In `internal/render/templates/Dockerfile.tmpl`, after line 69 (`RUN npm install -g @anthropic-ai/claude-code`), add a blank line, then:
   ```
   # --- Codex CLI ---
   RUN npm install -g @openai/codex
   ```
   This places it after the Claude Code install (line 69) and before the `USER root` directive (currently line 71). Both installs run as the `node` user since they follow the `USER node` directive at line 48. No `USER root` is needed because npm global installs into the node user's npm prefix.

2. **Update custom stage comment** -- In `internal/render/templates/custom-stage.tmpl`, line 19, change:
   ```
   # The agentbox stage above provides: mise runtimes, LSP servers, Claude Code,
   ```
   to:
   ```
   # The agentbox stage above provides: mise runtimes, LSP servers, Claude Code, Codex CLI,
   ```

3. **Add `TestDockerfile_CodexCLIInstall` test** -- New test in `internal/render/dockerfile_test.go`. Uses `Merge([]stack.StackID{stack.Go}, nil)` and asserts `strings.Contains(out, "npm install -g @openai/codex")`. Mirrors the existing `TestDockerfile_ClaudeCodeInstall` pattern.

4. **Add `TestDockerfile_CodexCLIInstall_EmptyConfig` assertion** -- In the existing `TestDockerfile_EmptyConfig` test, add an assertion that the Codex install appears even with no stacks (same as the existing Claude Code check). This verifies Codex is always installed regardless of stack selection.

5. **Add `TestDockerfile_DirectConfig_MinimalValid` assertion** -- In the existing `TestDockerfile_DirectConfig_MinimalValid` test, add an assertion for `npm install -g @openai/codex`.

6. **Add `TestDockerfile_CodexCLI_Ordering` test** -- New test asserting ordering: Claude Code install appears before Codex install, and Codex install appears before the `USER root` directive. Uses `strings.Index` comparisons (same pattern as `TestDockerfile_DevTools_OrderingInDockerfile`).

7. **Update `TestDockerfile_DevTools_OrderingInDockerfile`** -- The existing test asserts `golangci-lint` appears before Claude Code install. Now we also need to verify dev tools appear before both Claude Code and Codex. Add a `codexIdx` variable and assert `golangciIdx < codexIdx`.

8. **Update `TestCustomStage_ContainsHelpfulComments`** -- Add an assertion that the custom stage output contains `"Codex CLI"`.

### Testing Strategy

**New tests:**
- `TestDockerfile_CodexCLIInstall` -- Asserts `@openai/codex` appears in rendered Dockerfile output (via Merge + render path).
- `TestDockerfile_CodexCLI_Ordering` -- Asserts ordering: Claude Code < Codex < USER root (using `strings.Index` comparisons).

**Updated existing tests:**
- `TestDockerfile_EmptyConfig` -- Add `npm install -g @openai/codex` check (Codex always present, like Claude Code).
- `TestDockerfile_DirectConfig_MinimalValid` -- Add `npm install -g @openai/codex` check.
- `TestDockerfile_DevTools_OrderingInDockerfile` -- Add Codex ordering assertion (dev tools before Codex).
- `TestCustomStage_ContainsHelpfulComments` -- Add `"Codex CLI"` assertion.

**Existing tests that pass without change:**
- `TestDockerfile_ClaudeCodeInstall` -- Unchanged; Claude Code install is untouched.
- `TestDockerfile_NoTrailingWhitespace` -- Will pass; new lines have no trailing whitespace.
- `TestDockerfile_NoTemplateArtifacts` -- Will pass; no template actions added.
- `TestDockerfile_Deterministic` -- Will pass; no non-deterministic elements added.
- `TestDockerfile_DevTools_NoTripleNewlines` -- Will pass; blank line separation follows existing pattern (one blank line between sections).
- `TestDockerfile_UserAndWorkdir` -- Will pass; WORKDIR is still the last non-empty line.
- All integration tests -- Will pass; no file manifest changes, no new files.

### Open Questions

None. This is a straightforward static template addition with no design decisions.

## Definition of Done

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [x] Branch pushed to remote
- [x] PR created
- [x] Automated code review passed via `@review-backend` subagent (via Task tool)
- [x] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [x] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes) — N/A, no architectural changes
- [x] All other checklist items above are completed
- [x] User notified for human review

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-08 |
| challenge | completed | 1 | 2026-04-08 |
| implement | complete | 1 | 2026-04-08 |
| pr | completed | 1 | 2026-04-08 |
| review | completed | 2 | 2026-04-08 |
| codify | completed | 1 | 2026-04-08 |
