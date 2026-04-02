---
name: rework
description: Rework PR review feedback. Reads review comments from the open PR, fixes CRITICAL and WARNING findings, pushes fixes, then re-triggers review. Auto-detects the bean from the current branch.
metadata:
  argument-hint: "[optional: additional feedback or bean-id override]"
---

# Rework PR Review Feedback

Launch the `@rework` subagent to address review feedback on the current PR.

## Prerequisites

- You must be on a feature branch (not main)
- A PR must exist for the current branch
- Review comments must exist on the PR

## How to Launch

### Step 1: Detect the Bean ID

```bash
BRANCH=$(git branch --show-current)
BEAN_ID=$(echo "$BRANCH" | sed 's|^[^/]*/||' | grep -oP '^ccbox-\w+')
```

### Step 2: Launch the Rework Agent

```
Task tool call:
  subagent_type: "rework"
  description: "Rework PR review feedback"
  prompt: "Rework review feedback for bean <BEAN_ID>. Read the PR review comments, address all CRITICAL and WARNING findings, then push the fixes."
```

### Step 3: After the Agent Completes

1. **Summarize** — Report what was fixed
2. **Re-trigger review** — Launch `@review-backend`
3. **Report results**
