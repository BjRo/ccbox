---
# agentbox-6o49
title: Add Codex firewall domains to AlwaysOn allowlist
status: todo
type: task
created_at: 2026-04-08T09:16:26Z
updated_at: 2026-04-08T09:16:26Z
parent: agentbox-cqi5
---

Add OpenAI API and auth domains to the AlwaysOn firewall allowlist so Codex CLI can function inside network-isolated containers.

## Scope

### `internal/firewall/firewall.go`
Add to AlwaysOn domains:
```go
{Name: "api.openai.com", Category: Dynamic, Rationale: "OpenAI API - required for Codex CLI to function"},
{Name: "auth.openai.com", Category: Dynamic, Rationale: "OpenAI auth - required for Codex ChatGPT login flow"},
```

### Open questions to resolve during implementation
- **Auth domains**: Verify whether `auth.openai.com` or `auth0.openai.com` is used for ChatGPT login flow, or if it's handled through `api.openai.com`. Check Codex source (`codex-rs/login/`) for definitive list.
- **Telemetry domains**: Claude has `sentry.io` and `statsig.com`. Investigate whether Codex has its own telemetry endpoints that need allowlisting. Check `codex-rs/` source for telemetry/analytics URLs.

### Tests
- `internal/firewall/firewall_test.go`:
  - Assert `api.openai.com` appears in AlwaysOn domains
  - Assert auth domain(s) appear in AlwaysOn domains
  - Update any structural/count-based assertions

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
