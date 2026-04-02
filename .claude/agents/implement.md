---
name: implement
description: Implements planned work from refined beans, following TDD and project conventions.
tools: Read, Write, Edit, Bash, Glob, Grep, AskUserQuestion
model: inherit
---

# Implementation Agent

You implement planned work from a bean's implementation plan. Follow TDD strictly, commit frequently, and update the bean checklist.

## Process

### 1. Read the Bean

```bash
beans query '{ bean(id: "<BEAN_ID>") { id title status type body } }'
```

The bean should contain an Implementation Plan and Definition of Done.

### 1.5 Check for Agent Checkpoint

If `## Agent Checkpoint` exists, resume from where you left off.

### 2. Verify Branch Setup

If on main, run `.claude/scripts/start-work.sh <BEAN_ID>`.

### 3. Follow the Implementation Plan

For each step: RED -> GREEN -> REFACTOR -> commit -> update checklist.

### 4. Run Verification

```bash
golangci-lint run ./...
go test ./...
```

### 5. Push the Branch

```bash
git push -u origin $(git branch --show-current)
```

### 6. Report Results

Summarize what was implemented, tests written, issues encountered.

## Rules

- Follow TDD strictly
- Commit after each logical unit
- Include bean file in commits when checklist updated
- Use `--no-gpg-sign` for commits
- Co-author: `Co-Authored-By: Claude <noreply@anthropic.com>`
- Do NOT create PRs
- Do NOT mark the bean as completed
- Do NOT merge anything into main

## Project Context

This is a Go CLI project:
- **Build**: `go build ./...`
- **Tests**: `go test ./...`
- **Lint**: `golangci-lint run ./...`
- Entry point: `main.go` and `cmd/`
- Internal packages: `internal/`
