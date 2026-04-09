---
# agentbox-31g0
title: Update root README.md for Codex CLI support
status: refined
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

## Implementation Plan

### Approach

This is a documentation-only change to `/workspace/README.md`. No Go source code or templates are modified. There are no existing tests that validate the root README content (the `readme_test.go` tests validate the *generated per-project* README template, not the root README). Since this is a prose-only documentation update with no testable behavior, no new Go tests are needed.

The changes are organized into seven localized edits within the single file, each corresponding to one item in the bean description.

### Files to Modify

- `/workspace/README.md` — All seven documentation updates described below

### Steps

**1. Intro paragraph and "Why" section (lines 1-9) — mention Codex CLI alongside Claude Code**

- Line 1 description: Change from "running Claude Code in sandboxed environments" to "running Claude Code and Codex CLI in sandboxed environments" (or similar phrasing).
- Line 7 (Why section, first paragraph): Update "Claude Code works best with full permissions" to mention that "Claude Code and Codex CLI work best with full permissions".
- Line 9: Update the description to mention both tools. Currently says "gives Claude Code full permissions inside a network-isolated Docker container" — update to "gives Claude Code and Codex CLI full permissions...". Also update "ensures Claude Code can only reach explicitly approved domains" to "ensures the coding tools can only reach explicitly approved domains" (or name both).

**2. Features list (lines 12-20) — mention Codex CLI**

- Line 18 ("Claude Code settings sync"): Expand to mention both tools. Change to something like: "**Settings sync** -- Copies host Claude Code settings with jq deep-merge; copies Codex CLI config on first run"
- Line 19 ("LSP plugin configuration"): No change needed (LSP plugins are Claude Code specific and the description is fine as-is).

**3. Quick Start output (line 55) — update file count**

- The example output says "8 files" but agentbox now generates 11 files. Update to "11 files".

**4. Dockerfile section (lines 99-107) — add @openai/codex and bubblewrap**

- Line 105: Currently says `**Claude Code** via npm install -g @anthropic-ai/claude-code`. Change to: `**Claude Code and Codex CLI** via npm install -g @anthropic-ai/claude-code @openai/codex`
- Add `bubblewrap` to the system package list on line 107 ("Firewall tooling") or as a separate bullet. Looking at the actual Dockerfile template (line 19), bubblewrap is already in the apt-get install line alongside build-essential. Add a bullet: `**Sandbox runtime**: bubblewrap (required by Codex CLI sandbox mode)` or add it to the "Developer experience" bullet.

**5. Firewall always-on table (lines 141-151) — add OpenAI domains**

- The table currently has 5 rows. Add 5 new rows after the `*.anthropic.com` entry to match `firewall.go` (lines 70-74):

| `api.openai.com` | dynamic | OpenAI API for Codex CLI |
| `auth.openai.com` | dynamic | OpenAI auth for Codex ChatGPT login flow |
| `auth0.openai.com` | dynamic | OpenAI auth0 for Codex ChatGPT token refresh |
| `chatgpt.com` | dynamic | ChatGPT for Codex ChatGPT login auth flow |
| `accounts.openai.com` | dynamic | OpenAI accounts for Codex ChatGPT auth |

- Note: In the source `firewall.go`, all OpenAI domains use `Category: Dynamic`, not `Static`. The table column heading says "Category" so use "dynamic" consistently.
- **ALSO fix stale categories**: `github.com` and `api.github.com` are currently listed as "static" in the README but are `Dynamic` in `firewall.go` (lines 67-68). Change both to "dynamic" while editing this table.

**6. Settings sync section (lines 153-155) — document Codex copy-on-first-run**

- Currently only documents `sync-claude-settings.sh`. Add a new paragraph describing `sync-codex-settings.sh`:
  - "sync-codex-settings.sh copies the generated codex-config.toml into ~/.codex/config.toml inside the container. On first run it creates the file; on subsequent runs it skips the copy to preserve any manual changes (copy-on-first-run strategy, unlike the Claude Code deep-merge approach)."
- Reference the actual behavior from `/workspace/.devcontainer/sync-codex-settings.sh`: it checks `[ ! -f "$SETTINGS_FILE" ]` and only copies if the file does not exist.

**7. devcontainer.json section (lines 159-164) — add containerEnv, Codex volume mount, Codex extension**

- Add a bullet for `**containerEnv**` that forwards `OPENAI_API_KEY` from the host (matches template line 27-28: `"OPENAI_API_KEY": "${localEnv:OPENAI_API_KEY}"`).
- Update the **Mounts** bullet to mention the Codex config volume: "bash history, Claude config, **Codex config**, GitHub CLI config, and gitconfig" (matches template line 22: `agentbox-codex-config` volume).
- Update the **postStartCommand** bullet to mention Codex settings sync: "chains settings sync (Claude Code and Codex CLI) and firewall initialization" (matches template line 29 which calls both `sync-claude-settings.sh` and `sync-codex-settings.sh`).
- Note: The VS Code extension `openai.chatgpt` is listed in the devcontainer.json template (line 11). Mention it under customizations if there is a natural place, or add a note about the Codex extension being auto-configured.

**8. Generated files table (lines 170-181) — add 3 missing files**

- The table currently lists 8 files but agentbox generates 11. Add the 3 missing entries:
  - `codex-config.toml` — Codex CLI settings with full-auto approval policy and sandbox mode
  - `sync-codex-settings.sh` — Copies Codex CLI settings into the container (first-run only)
  - `mise-config.toml` — Runtime version configuration for mise (Go, Node, etc.)
- Also update the Dockerfile description to mention Codex CLI: "Container image with runtimes, LSPs, Claude Code, Codex CLI, and firewall tooling"

**9. Go version prerequisite (line 191) — evaluate change**

- The bean says to change "Go 1.25+" to "Go 1.24+". However, `go.mod` declares `go 1.25.0`, which means Go 1.25+ is the actual minimum. Keep "Go 1.25+" as-is unless the `go.mod` directive is also being changed. This item should be skipped or confirmed with the user.

**10. Firewall diagram (lines 113-135) — add Codex settings sync**

- The ASCII diagram at line 122 shows `sync-claude-settings.sh` in the startup flow. The actual `postStartCommand` (from devcontainer.json template line 29) now also calls `sync-codex-settings.sh`. Update the diagram to show both sync steps:
  ```
  sync-claude-settings.sh
            |
  sync-codex-settings.sh
            |
      init-firewall.sh
  ```
- Also update the diagram's bottom label from "Claude Code ready" to "Claude Code and Codex CLI ready" for consistency with the rest of the README updates.

### Testing Strategy

- **No new Go tests needed.** The root README.md is a hand-written documentation file, not a generated template. There are no existing tests that validate its content, and adding tests for static prose documentation would not provide meaningful value.
- **Verification**: After making edits, visually review the Markdown renders correctly (headings, tables, code blocks). Run `go test ./...` and `golangci-lint run ./...` to confirm no existing tests were broken (they should not be, since no Go files are modified).
- **Cross-check**: Verify every fact in the updated README against the actual source code:
  - Firewall domains against `/workspace/internal/firewall/firewall.go`
  - Generated file list against `/workspace/cmd/init.go` renderFiles map (lines 312-324)
  - devcontainer.json features against `/workspace/internal/render/templates/devcontainer.json.tmpl`
  - Dockerfile contents against `/workspace/internal/render/templates/Dockerfile.tmpl`
  - Codex sync behavior against `/workspace/.devcontainer/sync-codex-settings.sh`

### Open Questions

1. **Go version (item 7)**: The bean says to change "Go 1.25+" to "Go 1.24+" but `go.mod` declares `go 1.25.0`. Changing the README to "1.24+" would be incorrect since the project actually requires Go 1.25+. Should we skip this item, or is a `go.mod` change also planned?

## Definition of Done

- [x] Tests written (TDD: write tests before implementation) -- N/A: documentation-only change, no testable behavior
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [x] Branch pushed to remote
- [x] PR created — https://github.com/BjRo/agentbox/pull/38
- [x] Automated code review passed via `@review-backend` subagent (via Task tool)
- [x] Review feedback worked in via `/rework` and pushed to remote (if applicable) — N/A, clean review
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes) — N/A, no architectural changes
- [x] All other checklist items above are completed
- [x] User notified for human review

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | done | 1 | 2026-04-09 |
| challenge | done | 1 | 2026-04-09 |
| implement | done | 1 | 2026-04-09 |
| pr | done | 1 | 2026-04-09 |
| review | done | 1 | 2026-04-09 |
| codify | done | 1 | 2026-04-09 |
