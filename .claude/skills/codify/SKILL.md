---
name: codify
description: Extract reusable learnings from completed work into project documentation. Use with a bean ID argument, e.g. /codify ccbox-abc1
argument-hint: <bean-id>
context: fork
agent: codify
---

Codify learnings from bean $ARGUMENTS. Read the bean body and git diff against main. Identify new patterns, conventions, or architectural decisions. Check existing documentation for duplicates. Write findings to CLAUDE.md or decisions/. Commit and push if anything was written. Report what was codified or "No new patterns to codify" if nothing new was found.
