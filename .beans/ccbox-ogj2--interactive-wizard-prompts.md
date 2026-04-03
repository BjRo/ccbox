---
# ccbox-ogj2
title: Interactive wizard prompts
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:34:37Z
updated_at: 2026-04-03T07:21:09Z
parent: ccbox-puuq
---

## Description
Implement the interactive wizard flow for `ccbox init` (default when no flags given):

1. **Scan & confirm stacks**: Show detected stacks, let user toggle on/off, offer to add undetected ones
2. **Extra domains**: Ask if user wants to add custom domains to the firewall allowlist
3. **Confirmation**: Show summary of what will be generated, confirm before writing

Use a Go TUI library (e.g., charmbracelet/huh or bubbletea) for the prompts. Should feel polished and modern.

## Checklist

- [ ] Tests written (TDD)
- [ ] No TODO/FIXME/HACK/XXX comments
- [ ] Lint passes
- [ ] Tests pass
- [ ] Branch pushed
- [ ] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] All other checklist items

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | pending | | |
| challenge | pending | | |
| implement | pending | | |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |