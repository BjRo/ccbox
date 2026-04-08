---
# agentbox-5x61
title: Install golangci-lint in generated Dockerfile for Go stack
status: in-progress
type: feature
priority: normal
created_at: 2026-04-07T19:30:05Z
updated_at: 2026-04-08T06:26:42Z
---

When Go is a selected stack, the generated Dockerfile should install golangci-lint automatically. This is a standard Go development tool that every Go project needs. Currently it has to be installed manually after container creation and doesn't survive rebuilds.

The Go stack metadata in internal/stack/stack.go should include golangci-lint as a dev tool, and the Dockerfile template should install it (via go install from proxy.golang.org, since GitHub release downloads may be blocked by the firewall).

Also update this project's .devcontainer/ to include it.

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
