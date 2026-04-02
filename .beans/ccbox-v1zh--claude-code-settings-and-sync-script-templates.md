---
# ccbox-v1zh
title: Claude Code settings and sync script templates
status: in-progress
type: task
priority: high
created_at: 2026-04-02T10:35:16Z
updated_at: 2026-04-02T16:09:03Z
parent: ccbox-6z26
---

## Description
Two templates:

**claude-user-settings.json:**
```json
{
  "permissions": {
    "defaultMode": "bypassPermissions",
    "allow": ["Bash", "Read", "Write", "Edit", "Grep", "Glob", "Task", "WebFetch", "WebSearch"]
  },
  "enabledPlugins": [<stack-specific LSP plugins>]
}
```
The `enabledPlugins` array is parameterized: include `gopls-lsp` for Go, `typescript-lsp` for Node/TS, etc.

**sync-claude-settings.sh:**
Port the credfolio2 version as-is. This script:
1. Checks for template at `/workspace/.devcontainer/claude-user-settings.json`
2. If no existing settings: copies template
3. If settings exist: deep-merges with `jq -s '.[0] * .[1]'` (template wins)
4. Preserves runtime state written by Claude Code

This script is stack-agnostic — no parameterization needed, emit as static file.

## Checklist

- [ ] Tests written (TDD)
- [ ] No TODO/FIXME/HACK/XXX comments
- [ ] Lint passes
- [ ] Tests pass
- [ ] Branch pushed
- [ ] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] ADR written (if architectural changes)
- [ ] All checklist items completed
- [ ] User notified

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | pending | | |
| challenge | pending | | |
| implement | pending | | |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |