---
# agentbox-xgwq
title: Fix Codex CLI setup issues in generated containers
status: todo
type: bug
priority: normal
created_at: 2026-04-08T12:14:48Z
updated_at: 2026-04-08T12:14:54Z
parent: agentbox-cqi5
---

Two issues found when running Codex CLI in a generated devcontainer:

1. **Missing bubblewrap**: Warning 'could not find system bubblewrap on PATH'. Codex uses it for sandboxing. Since we set sandbox_mode=danger-full-access the container is the sandbox, but the warning is noisy. Fix: add `bubblewrap` to apt-get install in Dockerfile.tmpl.

2. **codex_apps MCP timeout**: Warning 'MCP client for codex_apps timed out after 30 seconds'. The built-in MCP server for web apps fails to start — likely needs additional allowlisted domains or should be disabled in the generated config. Fix: either add `startup_timeout_sec` to codex-config.toml.tmpl, disable the MCP server, or allowlist the required domains.

Also update the project's own .devcontainer to match.

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
