---
name: challenge
description: Launch the challenge subagent to stress-test a refined bean's implementation plan. Use after /refine to catch issues before implementation. Pass a bean ID as argument, e.g. /challenge ccbox-abc1
metadata:
  argument-hint: <bean-id>
---

# Challenge a Bean's Implementation Plan

Launch the `@challenge` subagent to stress-test a refined plan from the perspective of a staff-level Go engineer.

## How to Launch

```
Task tool call:
  subagent_type: "challenge"
  description: "Challenge <bean-title>"
  prompt: "Challenge the implementation plan for bean <BEAN_ID>. Read the bean, evaluate the plan against the Go engineer persona's checklist. Append a Challenge Report to the bean."
```

## What the Agent Does

- Reads the bean and its implementation plan
- Evaluates against the Go Engineer persona checklist
- Flags risks with severity: CRITICAL, WARNING, SUGGESTION
- Appends a Challenge Report with findings and verdict (APPROVED / NEEDS REVISION)

## What the Agent Does NOT Do

- Modify source code
- Modify the Implementation Plan section
