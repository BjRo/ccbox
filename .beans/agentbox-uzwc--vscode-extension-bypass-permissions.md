---
# agentbox-uzwc
title: Add VS Code extension bypass permissions to devcontainer.json
status: done
type: task
priority: high
created_at: 2026-04-07T12:00:00Z
updated_at: 2026-04-07T12:00:00Z
parent: agentbox-el52
---

The generated `devcontainer.json` only configures permissions via `claude-user-settings.json` (synced to `~/.claude/settings.json`). The VS Code extension has its own permission gate and won't offer bypass mode unless its own settings allow it.

## Problem

Two independent systems control Claude Code permissions:
1. **CLI/core**: `~/.claude/settings.json` → `permissions.defaultMode` (already configured via `claude-user-settings.json`)
2. **VS Code extension**: `devcontainer.json` → `customizations.vscode.settings` (not configured)

Without the VS Code extension settings, users running Claude Code via the VS Code extension inside the devcontainer still get permission prompts despite the CLI settings being set to bypass.

## Solution

Add VS Code extension settings to the `devcontainer.json` template:

```json
"customizations": {
  "vscode": {
    "extensions": ["anthropic.claude-code"],
    "settings": {
      "claude-code.initialPermissionMode": "bypassPermissions",
      "claude-code.allowDangerouslySkipPermissions": true
    }
  }
}
```

Keep `claude-user-settings.json` as the source of truth for permission *rules* (allow/deny lists). The VS Code settings only unlock bypass mode in the extension UI.

## Checklist

- [x] Update `devcontainer.json.tmpl` to include `customizations.vscode.settings`
- [x] Update template tests for new JSON structure
- [x] Regenerate `.devcontainer/` for this repo
- [x] Verify JSON validity in tests

## Implementation Plan

### Approach

This is a small, surgical change to a single static template file and its tests. The `devcontainer.json.tmpl` template is currently a static JSON document with zero Go template actions. It will remain static after this change -- the two new VS Code settings are hardcoded values, not derived from `GenerationConfig`. The template's static nature is already guarded by `TestDevContainer_IsStatic`, which renders with different configs and asserts byte-identical output; this test will continue to pass without modification.

### Files to Create/Modify

1. **`internal/render/templates/devcontainer.json.tmpl`** -- Add `settings` object inside `customizations.vscode` with the two bypass permission keys.

2. **`internal/render/devcontainer_test.go`** -- Update `TestDevContainer_FixedStructure` to assert the new `settings` sub-object exists with correct keys and values. No new test functions needed; the existing test structure already validates JSON shape.

3. **`.devcontainer/devcontainer.json`** -- Regenerate this project's own devcontainer by running `agentbox init` (or manually add the same two settings keys, since the template is static). This is the "eat your own dogfood" step listed in the checklist.

### Steps

1. **Update the template** -- Edit `internal/render/templates/devcontainer.json.tmpl` to add a `settings` key as a sibling to `extensions` inside `customizations.vscode`:

   Current structure:
   ```json
   "customizations": {
     "vscode": {
       "extensions": [
         "anthropic.claude-code"
       ]
     }
   }
   ```

   Target structure:
   ```json
   "customizations": {
     "vscode": {
       "extensions": [
         "anthropic.claude-code"
       ],
       "settings": {
         "claude-code.initialPermissionMode": "bypassPermissions",
         "claude-code.allowDangerouslySkipPermissions": true
       }
     }
   }
   ```

   Notes:
   - Add a comma after the closing `]` of `extensions` to separate it from `settings`.
   - The `settings` object uses two keys: one string value and one boolean value.
   - Keep the existing indentation style (2-space indent).

2. **Update `TestDevContainer_FixedStructure`** -- In `internal/render/devcontainer_test.go`, after the existing `extensions` assertions (line ~79), add assertions for the new `settings` sub-object:
   - Assert `vscode["settings"]` exists and is `map[string]any`.
   - Assert `settings["claude-code.initialPermissionMode"]` equals `"bypassPermissions"` (string).
   - Assert `settings["claude-code.allowDangerouslySkipPermissions"]` equals `true` (bool; JSON unmarshal into `any` produces `bool` for JSON booleans).

3. **Verify existing tests still pass** -- The following existing tests require no changes but must pass:
   - `TestDevContainer_ValidJSON` -- Validates JSON syntax; will pass because the template remains valid JSON.
   - `TestDevContainer_IsStatic` -- Renders with different configs and asserts byte-identical output; will pass because the new settings are hardcoded (no template actions).
   - `TestDevContainer_Deterministic` -- Renders the same config twice and asserts byte-equality; will pass trivially.
   - `TestDevContainer_EmptyConfig` -- Renders with zero-value `GenerationConfig`; will pass because the new content is static.
   - `TestDevContainer_MountsContent` -- Tests mount entries; unaffected by customization changes.

4. **Regenerate `.devcontainer/devcontainer.json` for this repo** -- Since the template is static, the simplest approach is to copy the updated template content directly into `.devcontainer/devcontainer.json`. Alternatively, delete `.devcontainer/` and run `go run . init` to regenerate. The result is the same because the template contains no dynamic actions.

5. **Run the full test suite** -- Execute `go test ./...` (unit tests) and `go test -tags integration ./...` (integration tests). The integration tests in `cmd/init_integration_test.go` parse the generated `devcontainer.json` as JSON and check structural fields; they will pass because the output remains valid JSON. No integration test changes are needed -- those tests do not assert the absence of `settings` or check an exact field count on the `vscode` object.

6. **Run the linter** -- Execute `golangci-lint run ./...` to confirm no issues.

### Testing Strategy

- **No new test functions needed.** The existing test suite already covers:
  - JSON validity (`TestDevContainer_ValidJSON`)
  - Structural field checks (`TestDevContainer_FixedStructure` -- updated in step 2)
  - Static template invariant (`TestDevContainer_IsStatic`)
  - Determinism (`TestDevContainer_Deterministic`)
  - Empty config safety (`TestDevContainer_EmptyConfig`)
  - Integration end-to-end (`TestIntegration_SingleGoStack`, `TestIntegration_MultiStack`)

- **What the updated `TestDevContainer_FixedStructure` verifies:**
  - `customizations.vscode.settings` exists as a JSON object.
  - `claude-code.initialPermissionMode` is the string `"bypassPermissions"`.
  - `claude-code.allowDangerouslySkipPermissions` is the boolean `true`.

### Open Questions

None. The VS Code extension setting names and values are specified in the bean description and match the documented Claude Code VS Code extension configuration. The change is purely additive to a static template, so there is no risk of breaking existing functionality.
