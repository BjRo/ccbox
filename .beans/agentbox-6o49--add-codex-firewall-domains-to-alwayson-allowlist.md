---
# agentbox-6o49
title: Add Codex firewall domains to AlwaysOn allowlist
status: in-progress
type: task
priority: normal
created_at: 2026-04-08T09:16:26Z
updated_at: 2026-04-08T11:01:24Z
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

### Tests
- `internal/firewall/firewall_test.go`:
  - Assert `api.openai.com` appears in AlwaysOn domains
  - Assert `auth.openai.com` appears in AlwaysOn domains
  - Update the hardcoded `expected` map in `TestRegistry_AlwaysOnDomains`

## Implementation Plan

### Approach

Add two new domains to the AlwaysOn allowlist in the firewall registry. This is a data-only change to a single file (`internal/firewall/firewall.go`) plus test updates in `internal/firewall/firewall_test.go`. No new packages, no new templates, no architectural changes.

Both domains are categorized as `Dynamic` because the OpenAI API and auth endpoints sit behind load balancers/CDNs with rotating IPs, consistent with how the existing `*.anthropic.com` and `github.com` entries are categorized.

### Resolved Open Questions

**Auth domains**: Use `auth.openai.com`. The Codex CLI login flow (OAuth device authorization grant) directs the user's browser to `auth.openai.com` for the ChatGPT login, then polls `api.openai.com` for the resulting token. The `auth.openai.com` subdomain is the correct endpoint -- not `auth0.openai.com`, which is an older endpoint that redirects to `auth.openai.com`. Including `auth.openai.com` ensures the ChatGPT login flow works. When users authenticate via `OPENAI_API_KEY` instead, only `api.openai.com` is needed, but `auth.openai.com` is harmless in that case.

**Telemetry domains**: No additional telemetry domains are needed. Unlike Claude Code (which uses third-party services `sentry.io` and `statsig.com`), the Codex CLI routes telemetry through OpenAI's own infrastructure at `api.openai.com`. No separate telemetry endpoints need allowlisting.

**Wildcard vs specific subdomains**: Use specific subdomains (`api.openai.com`, `auth.openai.com`) rather than a `*.openai.com` wildcard. This follows the principle of least privilege -- the wildcard would allow access to `chat.openai.com`, `platform.openai.com`, and other services not needed for CLI operation. The existing Claude entry uses `*.anthropic.com` because Claude Code contacts multiple Anthropic subdomains; Codex needs only two.

### Files to Modify

- `internal/firewall/firewall.go` -- Add 2 entries to the AlwaysOn Domains slice
- `internal/firewall/firewall_test.go` -- Update `TestRegistry_AlwaysOnDomains` expected map and add merge spot-checks

### Steps

1. **Update the AlwaysOn registry entry** -- `internal/firewall/firewall.go`, lines 63-73
   - Add `{Name: "api.openai.com", Category: Dynamic, Rationale: "OpenAI API - required for Codex CLI to function"}` after the existing Anthropic entry
   - Add `{Name: "auth.openai.com", Category: Dynamic, Rationale: "OpenAI auth - required for Codex ChatGPT login flow"}` after `api.openai.com`
   - Both use `Dynamic` category (CDN/load-balanced endpoints)

2. **Update `TestRegistry_AlwaysOnDomains`** -- `internal/firewall/firewall_test.go`, lines 23-56
   - Add `"api.openai.com": Dynamic` to the `expected` map
   - Add `"auth.openai.com": Dynamic` to the `expected` map
   - The count assertion (`len(al.Domains) != len(expected)`) auto-adjusts since it compares against the map length

3. **Add merge spot-checks** -- `internal/firewall/merge_test.go`, `TestMerge_AlwaysOnIncluded` (around line 77)
   - Add assertions that `api.openai.com` is in Dynamic (alongside existing `github.com` and `*.anthropic.com` spot-checks)

4. **Verify all downstream tests pass without changes**
   - `internal/firewall/merge_test.go`: `collectExpected` reads from the registry dynamically -- structural assertions auto-adjust
   - `internal/render/firewall_test.go`: Structural assertions via `Merge` -- new domains flow through automatically
   - `internal/render/readme_test.go`: Same structural pattern -- no hardcoded domain counts
   - Run `go test ./internal/firewall/...` and `go test ./internal/render/...`

### Testing Strategy

- **TDD sequence**: Write the test changes first (add `api.openai.com` and `auth.openai.com` to the expected map in `TestRegistry_AlwaysOnDomains`), verify the test fails (count mismatch: "AlwaysOn has 5 domains, want 7"), then add the registry entries
- **Spot-check in merge test**: Add assertion in `TestMerge_AlwaysOnIncluded` that `api.openai.com` appears in the Dynamic list (alongside existing `github.com` and `*.anthropic.com` spot-checks)
- **Full test sweep**: Run `go test ./internal/firewall/...` and `go test ./internal/render/...` to verify no structural assertions break
- **Lint**: Run `golangci-lint run ./...` to confirm no issues

### Open Questions

None remaining -- all resolved above.

## Definition of Done

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [x] Branch pushed to remote
- [ ] PR created
- [ ] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-08 |
| challenge | completed | 1 | 2026-04-08 |
| implement | complete | 1 | 2026-04-08 |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |
