---
name: challenge
description: Stress-test a refined bean's implementation plan from the Go engineer persona. Use after /refine. Pass a bean ID as argument, e.g. /challenge ccbox-abc1
argument-hint: <bean-id>
context: fork
agent: challenge
---

Challenge the implementation plan for bean $ARGUMENTS. Read the bean, evaluate the plan against the Go engineer persona's checklist. Append a Challenge Report with findings and verdict (APPROVED / NEEDS REVISION).
