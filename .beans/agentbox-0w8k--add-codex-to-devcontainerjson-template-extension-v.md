---
# agentbox-0w8k
title: Add Codex to devcontainer.json template (extension, volume, env, postStart)
status: in-progress
type: task
priority: normal
created_at: 2026-04-08T09:16:37Z
updated_at: 2026-04-08T11:24:09Z
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

## Implementation Plan

### Approach

The devcontainer.json template is currently fully static (no Go template actions) and will remain so after these changes. All four additions are static text changes to the JSON template. No new Go code is needed beyond the template edits. No `GenerationConfig` changes, no new FuncMap helpers, no new render functions. The `devcontainer.go` render function and the `DevContainer()` API remain unchanged.

The template will remain stack-agnostic: both Claude Code and Codex settings are always-on regardless of which stacks are selected (same design as the Dockerfile Codex install from agentbox-qr1p).

### Files to Modify

- `internal/render/templates/devcontainer.json.tmpl` -- Add extension, volume mount, containerEnv, and update postStartCommand
- `internal/render/devcontainer_test.go` -- Add tests for all four new elements, update existing mount count assertion from 4 to 5

### Steps

#### Step 1: Write tests first (TDD)

Add the following tests to `internal/render/devcontainer_test.go`:

1. **Update `TestDevContainer_FixedStructure`** -- This is the main structural test that already validates extensions, mounts, and postStartCommand. Updates needed:
   - Add assertion that `openai.chatgpt` appears in `extensions` array (alongside existing `anthropic.claude-code` check at line 76-83)
   - Update mount count assertion from `len(mounts) != 4` to `len(mounts) != 5` (line 142)
   - Add assertion that `containerEnv` field exists and contains `OPENAI_API_KEY` key
   - Add assertion that `postStartCommand` contains `sync-codex-settings.sh`
   - Add assertion that `postStartCommand` contains `/home/node/.codex` (in the chown portion)

2. **Update `TestDevContainer_MountsContent`** -- Add `agentbox-codex-config` to the `checks` slice (line 212-216) to verify Codex volume mount is present.

3. **Add `TestDevContainer_PostStartCommand_Ordering`** -- New test asserting the execution order within `postStartCommand`:
   - `chown` appears before `sync-claude-settings.sh`
   - `sync-claude-settings.sh` appears before `sync-codex-settings.sh`
   - `sync-codex-settings.sh` appears before `init-firewall.sh`
   Use `strings.Index` comparisons on the parsed `postStartCommand` string.

4. **Add `TestDevContainer_ContainerEnv_Structure`** -- New test that unmarshals devcontainer.json and validates:
   - `containerEnv` field exists and is a map
   - `OPENAI_API_KEY` key maps to `${localEnv:OPENAI_API_KEY}` value (exact match)

5. **Verify existing tests still pass** -- The following tests should continue to work without modification:
   - `TestDevContainer_ValidJSON` -- Still valid JSON after template edits
   - `TestDevContainer_EmptyConfig` -- Same structure present regardless of config
   - `TestDevContainer_IsStatic` -- Template remains static (byte-identical across different configs)
   - `TestDevContainer_Deterministic` -- No non-deterministic elements introduced

#### Step 2: Update `internal/render/templates/devcontainer.json.tmpl`

The current template (35 lines) needs four changes:

**2a. Add Codex extension** (line 10, extensions array):
Change:
```json
    "extensions": [
        "anthropic.claude-code"
    ],
```
To:
```json
    "extensions": [
        "anthropic.claude-code",
        "openai.chatgpt"
    ],
```

**2b. Add Codex volume mount** (line 21, mounts array):
Add a fifth entry after the `agentbox-claude-config` volume mount:
```json
    "source=agentbox-codex-config,target=/home/node/.codex,type=volume",
```
This should be placed after the `agentbox-claude-config` line (line 20) and before the `${localEnv:HOME}/.config/gh` bind mount (line 21). This groups the two volume mounts (bash history + Claude config + Codex config) together before the bind mounts.

**2c. Add containerEnv field** (new field):
Add a `containerEnv` object after the `mounts` array. Place it between `mounts` and `postStartCommand` for logical grouping:
```json
  "containerEnv": {
    "OPENAI_API_KEY": "${localEnv:OPENAI_API_KEY}"
  },
```
This uses the devcontainer `${localEnv:...}` syntax to forward the host environment variable. If the variable is not set on the host, the container gets an empty string (Codex falls back to ChatGPT interactive login in that case).

**2d. Update postStartCommand** (line 24):
Change from:
```json
  "postStartCommand": "sudo chown -R node:node /home/node/.claude /home/node/.bash_history_volume && bash .devcontainer/sync-claude-settings.sh && sudo bash .devcontainer/init-firewall.sh",
```
To:
```json
  "postStartCommand": "sudo chown -R node:node /home/node/.claude /home/node/.codex /home/node/.bash_history_volume && bash .devcontainer/sync-claude-settings.sh && bash .devcontainer/sync-codex-settings.sh && sudo bash .devcontainer/init-firewall.sh",
```
Changes:
- Add `/home/node/.codex` to the `chown` list (between `.claude` and `.bash_history_volume`)
- Add `bash .devcontainer/sync-codex-settings.sh` in the command chain (after `sync-claude-settings.sh`, before `init-firewall.sh`)

Note: `sync-codex-settings.sh` does NOT need `sudo` since it operates on user-owned directories (matching the `sync-claude-settings.sh` pattern).

#### Step 3: Verify all tests pass

Run `go test ./internal/render/...` to confirm all unit tests pass (both new and existing).
Run `go test -tags integration ./cmd/...` to confirm integration tests pass (the integration tests already assert devcontainer.json is valid JSON and check for `"target": "custom"`).
Run `golangci-lint run ./...` to confirm lint passes.

### Testing Strategy

**Updated existing tests:**
- `TestDevContainer_FixedStructure` -- Updated mount count (4 to 5), new extension assertion, new containerEnv assertion, new postStartCommand assertions
- `TestDevContainer_MountsContent` -- Add `agentbox-codex-config` check

**New tests:**
- `TestDevContainer_PostStartCommand_Ordering` -- Verify the ordering of commands in the postStartCommand string: chown < sync-claude < sync-codex < init-firewall
- `TestDevContainer_ContainerEnv_Structure` -- Verify containerEnv field structure and OPENAI_API_KEY value

**Existing tests that validate without change:**
- `TestDevContainer_ValidJSON` -- JSON validity remains (catches missing commas, trailing commas, etc.)
- `TestDevContainer_EmptyConfig` -- Structure present regardless of config
- `TestDevContainer_IsStatic` -- Template is still stack-agnostic (byte-identical across configs)
- `TestDevContainer_Deterministic` -- No non-deterministic elements

**Integration tests (no changes needed):**
- `TestIntegration_SingleGoStack` already checks `json.Unmarshal` validity and `"target": "custom"` -- these continue to pass
- Adding explicit Codex assertions to integration tests would be scope creep for this bean (tracked separately in agentbox-comb)

### Open Questions

None. All changes are well-defined static template additions following established patterns. The sync script (`sync-codex-settings.sh`) and config template (`codex-config.toml`) already exist from agentbox-vr4z. The firewall domains (`api.openai.com`, `auth.openai.com`) already exist from agentbox-6o49. The Codex CLI installation already exists from agentbox-qr1p. This bean wires those pieces together in the devcontainer.json template.

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
