---
# ccbox-ogj2
title: Interactive wizard prompts
status: todo
type: task
priority: normal
created_at: 2026-04-02T10:34:37Z
updated_at: 2026-04-02T10:34:37Z
parent: ccbox-puuq
---

## Description
Implement the interactive wizard flow for `ccbox init` (default when no flags given):

1. **Scan & confirm stacks**: Show detected stacks, let user toggle on/off, offer to add undetected ones
2. **Extra domains**: Ask if user wants to add custom domains to the firewall allowlist
3. **Confirmation**: Show summary of what will be generated, confirm before writing

Use a Go TUI library (e.g., charmbracelet/huh or bubbletea) for the prompts. Should feel polished and modern.