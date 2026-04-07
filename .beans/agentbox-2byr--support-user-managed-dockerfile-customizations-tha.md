---
# agentbox-2byr
title: Support user-managed Dockerfile customizations that survive regeneration
status: draft
type: feature
priority: normal
created_at: 2026-04-07T19:30:23Z
updated_at: 2026-04-07T19:30:28Z
---

Users need a way to add project-specific tools (e.g., beans CLI) to the devcontainer that won't be overwritten when agentbox regenerates files. Currently agentbox refuses to run if .devcontainer/ exists, and there's no update path at all.

Explore a Dockerfile.custom or similar mechanism where agentbox owns the base Dockerfile and the user owns an extension layer. The base Dockerfile should reference the custom file so both are used during build. Agentbox regeneration updates the base, leaves the custom file untouched.

This is a design exploration — needs /explore-approaches before implementation.

## Definition of Done

- [ ] Tests written (TDD: write tests before implementation)
- [ ] No new TODO/FIXME/HACK/XXX comments introduced
- [ ] `golangci-lint run ./...` passes with no errors
- [ ] `go test ./...` passes with no failures
- [ ] Branch pushed to remote
- [ ] PR created
- [ ] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review
