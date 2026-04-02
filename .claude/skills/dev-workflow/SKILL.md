---
name: dev-workflow
description: Development workflow for feature implementation. Use when starting work on any bean/task to ensure proper git hygiene, TDD, and PR-based review.
---

# Development Workflow

Follow this workflow for all feature development.

## Starting Work on a Bean

### Quick Start

```bash
.claude/scripts/start-work.sh <bean-id>
```

This automatically:
- Ensures main is up-to-date
- Derives branch name from bean type + title
- Creates feature branch
- Marks bean as in-progress
- Commits bean status change

### 4. Develop Using TDD

1. **RED**: Write a failing test
2. **GREEN**: Write minimum code to pass
3. **REFACTOR**: Clean up while green

Commit frequently with meaningful messages.

### 5. Update Bean Checklist

Check off items as you complete them. Include bean file in commits.

### 6. Push and Open Pull Request

```bash
git push -u origin <branch-name>
gh pr create --title "<type>: <description>" --body "$(cat <<'EOF'
## Summary
Brief description of changes.

## Bean
<bean-id>

## Test Plan
- All tests pass (`go test ./...`)
- Lint clean (`golangci-lint run ./...`)

Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

### 7. Run Automated Code Review

Launch `@review-backend` as a subagent via Task tool.

If CRITICAL or WARNING findings, use `/rework` to address them.

### 8. Wait for Human Review

Do NOT merge the PR yourself.

### 9. After Merge

```bash
.claude/scripts/post-merge.sh <bean-id>
```

## Mandatory Definition of Done

Every bean MUST include a "Definition of Done" checklist. See `.claude/templates/definition-of-done.md`.

## Rules

1. Never commit directly to main
2. Never merge your own PRs
3. Always use TDD
4. Always update bean checklists
