---
name: parallel-deliver
description: Launch parallel delivery pipelines for multiple beans. Each bean gets its own Claude CLI instance in an isolated worktree. Use with 1-3 bean IDs, e.g. /parallel-deliver ccbox-abc1 ccbox-xyz9
argument-hint: <bean-id> [bean-id...] (max 3)
---

# Parallel Deliver

Launch 1-3 beans through the full delivery pipeline in parallel. Each bean runs in its own headless Claude Code instance with an isolated git worktree.

## Architecture

```
This conversation (orchestrator)
├── launch-deliver.sh bean-1 --slot 1  (isolated worktree)
├── launch-deliver.sh bean-2 --slot 2  (isolated worktree)
└── launch-deliver.sh bean-3 --slot 3  (isolated worktree)
```

Each instance runs the `deliver` agent which orchestrates: refine -> challenge -> implement -> PR -> review -> rework -> codify.

## Prerequisites

- Beans must exist and have descriptions with enough context to refine
- You must be on main (each instance creates its own worktree and feature branch)

## How to Launch

### Step 1: Parse and Validate

Extract bean IDs from the arguments. Validate each:

```bash
# For each bean ID:
beans query '{ bean(id: "<BEAN_ID>") { id title status type } }' --json
```

- Verify the bean exists
- Verify it is not already `completed`
- Max 3 beans (slots 1-3)

### Step 2: Launch Instances

For each bean, launch a background delivery instance. Use the Bash tool with `run_in_background: true` for each:

```bash
.claude/scripts/launch-deliver.sh <BEAN_ID> --slot <N>
```

Launch all instances in parallel (multiple Bash tool calls in a single message).

Report to the user:

> Launched parallel delivery for N beans:
> | Bean | Slot | Worktree |
> |------|------|----------|
> | ccbox-abc1 | 1 | .claude/worktrees/deliver-ccbox-abc1 |
> | ccbox-xyz9 | 2 | .claude/worktrees/deliver-ccbox-xyz9 |
>
> Monitoring progress via bean Pipeline State...

### Step 3: Monitor and Report

As each background task completes (you will be notified automatically), check the bean's state:

```bash
# Read pipeline state
beans query '{ bean(id: "<BEAN_ID>") { body } }' --json | jq -r '.bean.body'
```

Look for:
- `## Pipeline State` section — parse the table for current phase and status
- `### Escalation` section — check if it says `(none)` or has content

Also check the log for any error output:

```bash
tail -30 .claude/logs/deliver-<BEAN_ID>.log
```

### Step 4: Handle Completions

**For each completed bean**, report to the user:

If all phases completed:
> Bean <BEAN_ID> delivered successfully!
> - PR: <PR_URL from log or gh pr view in worktree>
> - All review phases passed

If the instance escalated:
> Bean <BEAN_ID> needs attention:
> - Phase: <escalated phase>
> - Reason: <from Escalation section>
> - Action needed: <from Escalation section>

If the instance failed (non-zero exit, no escalation):
> Bean <BEAN_ID> failed unexpectedly.
> - Check log: .claude/logs/deliver-<BEAN_ID>.log
> - Worktree: .claude/worktrees/deliver-<BEAN_ID>

### Step 5: Handle Escalations

When a bean has escalated, present the escalation to the user and wait for guidance.

After the user resolves the issue (e.g., manually updates the bean plan), relaunch:

```bash
.claude/scripts/launch-deliver.sh <BEAN_ID> --slot <N> --resume
```

The `--resume` flag tells the new instance to check the Pipeline State and continue from where the previous one stopped.

### Step 6: Final Summary

When all beans are done (completed or escalated-and-resolved), report:

> ## Parallel Delivery Summary
>
> | Bean | Status | PR | Notes |
> |------|--------|----|-------|
> | ccbox-abc1 | Delivered | <PR_URL> | Clean reviews |
> | ccbox-xyz9 | Delivered | <PR_URL> | 1 rework cycle |
>
> All PRs are ready for human review.

## Checking Status Mid-Flight

If the user asks "what's the status?" at any point, check each bean:

```bash
for BEAN_ID in <list>; do
  echo "=== $BEAN_ID ==="
  beans query "{ bean(id: \"$BEAN_ID\") { title body } }" --json | jq -r '.bean.body' | grep -A20 '## Pipeline State' | head -15
  echo ""
done
```

## Resuming After Failure

If the entire orchestrator session was interrupted, the user can restart by running `/parallel-deliver` with `--resume`:

```
/parallel-deliver ccbox-abc1 ccbox-xyz9 --resume
```

In resume mode:
1. Check each bean's Pipeline State to see where it stopped
2. For beans with incomplete pipelines, relaunch with `--resume`
3. For beans already completed, skip them

## Cleaning Up

After all PRs are merged and beans completed, clean up worktrees:

```bash
# Remove a specific worktree
git worktree remove .claude/worktrees/deliver-<BEAN_ID>

# Remove all deliver worktrees
for wt in .claude/worktrees/deliver-*; do
  git worktree remove "$wt" 2>/dev/null || echo "Skipping $wt (may have changes)"
done
```

## Important Notes

- **Max 3 parallel instances** — limited by slots 1-3
- **Do NOT merge PRs** — always wait for human review
- **Do NOT mark beans as completed** — that happens after human merge via `/post-merge`
- **Each instance is independent** — if one fails, the others continue unaffected
- **Beans are the source of truth** — all progress is tracked in the bean's Pipeline State section
- **Logs are secondary** — check `.claude/logs/deliver-<BEAN_ID>.log` for debugging, but the bean state is canonical
