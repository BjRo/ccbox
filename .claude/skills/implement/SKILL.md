---
name: implement
description: Launch the implement subagent to execute a refined bean's implementation plan. Use with a bean ID argument, e.g. /implement ccbox-abc1
metadata:
  argument-hint: <bean-id>
---

# Implement a Bean

Launch the `@implement` subagent to execute a bean's implementation plan in an isolated context.

## Prerequisites

1. **Bean must have an implementation plan** — Run `/refine <bean-id>` first
2. **Bean must have a checklist** — The agent works through checklist items in order

## How to Launch

```
Task tool call:
  subagent_type: "implement"
  description: "Implement <bean-title>"
  prompt: "Implement bean <BEAN_ID>. Read the bean, follow its implementation plan using TDD, commit as you go, and update the bean checklist. Report what was completed when done."
```

## After the Agent Completes

1. **Summarize** — Report what the agent completed
2. **Create PR** — `gh pr create` with a summary
3. **Run review** — Launch `@review-backend` subagent
4. **Check off remaining DoD items**
5. **Report results** — Share PR link and review findings

## What the Agent Does

- Reads the bean and its implementation plan
- Sets up the feature branch (via `start-work.sh`) if not already on one
- Follows TDD: RED -> GREEN -> REFACTOR for each step
- Commits frequently with meaningful messages
- Updates bean checklist items as they're completed
- Runs `golangci-lint run ./...` and `go test ./...` at the end
- Pushes the branch to remote

## What the Agent Does NOT Do

- Create PRs
- Launch review agents
- Mark the bean as completed
- Merge anything into main
