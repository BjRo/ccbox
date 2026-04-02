---
name: codify
description: Extract reusable learnings from completed work into project documentation. Use with a bean ID argument, e.g. /codify ccbox-abc1
metadata:
  argument-hint: <bean-id>
---

# Codify Learnings

Launch the `@codify` subagent to extract reusable learnings into documentation.

## How to Launch

```
Task tool call:
  subagent_type: "codify"
  description: "Codify learnings from <bean-title>"
  prompt: "Codify learnings from bean <BEAN_ID>. Read the bean body and git diff against main. Identify new patterns, conventions, or decisions. Write to CLAUDE.md or decisions/. Commit and push."
```

## What the Agent Does

- Reads the bean body and git diff
- Identifies reusable learnings
- Checks existing documentation for duplicates
- Writes to CLAUDE.md or `decisions/` (ADR format)
- Commits and pushes

## What the Agent Does NOT Do

- Modify source code
- Remove existing documentation
- Create duplicates
