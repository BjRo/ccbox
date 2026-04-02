---
name: decision
description: Document a technical or process decision. Use after making significant changes to the tech stack, introducing new patterns, or deprecating existing approaches.
---

# Decision Documentation

## When This Skill Applies

- Adding or removing dependencies, frameworks, or tools
- Introducing new architectural patterns
- Deprecating existing approaches
- Making significant technical choices

## Process

### 1. Gather Information

If not clear from context, ask about:
- What was done?
- Why was it done?
- What bean introduced this?

### 2. Generate the Decision File

Create in `/decisions/` with naming: `YYYYMMDDHHMMSS-kebab-case-title.md`

### 3. Use This Template

```markdown
# [Title]

**Date**: YYYY-MM-DD
**Bean**: [bean-id or "N/A"]

## Context
[What situation led to this decision?]

## Decision
[What was decided and implemented?]

## Reasoning
[Why this approach? What alternatives were considered?]

## Consequences
[What are the implications going forward?]
```

### 4. Update the Decision Index

Add a new row to `/decisions/README.md` index table.

### 5. Commit Both Files

```bash
git add decisions/
git commit -m "docs: Add ADR for <title>"
```
