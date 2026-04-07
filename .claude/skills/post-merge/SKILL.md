---
name: post-merge
description: "Post-Merge Cleanup. Use with bean ID, e.g. /post-merge agentbox-abc1"
argument-hint: <bean-id>
---

# Post-Merge Cleanup

Clean up after a PR has been merged.

## Execution

```bash
.claude/scripts/post-merge.sh <BEAN_ID>
```

This switches to main, pulls, deletes the branch, marks the bean complete, and pushes.

## After Running

Report the summary: which beans completed, which branches cleaned up.
