---
# agentbox-96ja
title: Rename config.toml to mise-config.toml
status: todo
type: task
priority: normal
created_at: 2026-04-08T12:04:48Z
updated_at: 2026-04-08T12:04:55Z
parent: agentbox-cqi5
---

Now that codex-config.toml exists in .devcontainer/, the generic config.toml name is ambiguous. Rename it to mise-config.toml for clarity. Affects: Dockerfile.tmpl COPY directive, config.toml.tmpl filename, cmd/init.go and cmd/update.go file maps, the project's own .devcontainer/config.toml, and update command's config preservation logic.

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
