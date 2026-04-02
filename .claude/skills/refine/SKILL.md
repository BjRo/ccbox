---
name: refine
description: Launch the refine subagent to create a detailed implementation plan for a bean. Use with a bean ID argument, e.g. /refine ccbox-abc1
metadata:
  argument-hint: <bean-id>
---

# Refine a Bean

Launch the `@refine` subagent to develop a detailed, actionable implementation plan.

## How to Launch

```
Task tool call:
  subagent_type: "refine"
  description: "Refine <bean-title>"
  prompt: "Refine bean <BEAN_ID>. Read the bean, explore the codebase, and develop a detailed implementation plan. Ask clarifying questions if needed. Update the bean body with the plan."
```

## What the Agent Does

- Reads the bean and understands the requirements
- Explores the codebase to understand current state, patterns, and dependencies
- Asks clarifying questions via `AskUserQuestion` when ambiguous
- Writes a structured implementation plan (approach, files, steps, testing strategy)
- Updates the bean body with the plan (preserving existing content)

## What the Agent Does NOT Do

- Modify source code — planning agent only
- Mark the bean as completed
