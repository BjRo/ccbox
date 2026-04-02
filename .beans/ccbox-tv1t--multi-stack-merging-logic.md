---
# ccbox-tv1t
title: Multi-stack merging logic
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:34:25Z
updated_at: 2026-04-02T15:06:47Z
parent: ccbox-2n15
---

## Description
When multiple stacks are detected, merge their metadata:
- Combine all runtimes into a single mise.toml
- Merge LSP server installations in Dockerfile
- Merge LSP plugin lists in claude-user-settings.json
- Union all default + dynamic domain allowlists
- Deduplicate domains

Input: list of detected Stack objects + user-selected extras
Output: a merged `GenerationConfig` struct used by the template engine