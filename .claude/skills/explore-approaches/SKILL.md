---
name: explore-approaches
description: Research a problem space and propose 2-3 concrete approaches with tradeoffs before implementation. Pass a bean ID as argument, e.g. /explore-approaches ccbox-abc1
metadata:
  argument-hint: <bean-id>
---

# Explore Approaches for a Bean

Launch the `@explore-approaches` subagent to research a problem space and propose approaches.

## When to Use

- Complex template generation challenges
- Cross-platform compatibility issues
- Any problem where the right approach isn't obvious

## How to Launch

```
Task tool call:
  subagent_type: "explore-approaches"
  description: "Explore approaches for <bean-title>"
  prompt: "Explore approaches for bean <BEAN_ID>. Research the problem space, check for prior attempts, and propose 2-3 concrete approaches with tradeoffs. Append an Approach Exploration report to the bean."
```

## What the Agent Does

- Reads the bean and any existing plan
- Researches the execution context and constraints
- Checks git history for prior attempts
- Proposes 2-3 fundamentally different approaches
- Recommends one with reasoning
