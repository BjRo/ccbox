---
# ccbox-3wvl
title: Unit tests for template generation
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:36:00Z
updated_at: 2026-04-03T08:38:14Z
parent: ccbox-6g75
---

## Description
Tests that verify generated files are correct:
- Render Dockerfile for Go-only project → contains gopls, go runtime in mise, no node/python/ruby
- Render Dockerfile for multi-stack (Go + Node) → contains both runtimes, both LSPs
- Render devcontainer.json → valid JSON, has correct mounts/extensions/capabilities
- Render claude-user-settings.json for Go → has gopls-lsp plugin
- Render dynamic-domains.conf for Python → has pypi.org
- Render with extra domains → extra domains appear in dynamic-domains.conf
- All generated shell scripts have correct shebang and are syntactically valid (shellcheck if available)

Use golden file testing pattern: expected outputs checked into testdata/.

## Implementation Plan

### Situation Analysis

The `internal/render/` package already has 88 tests across 6 test files with 87.5% statement coverage. The Merge function has 97.9% coverage. The render functions (Dockerfile, DevContainer, RenderClaude, RenderFirewall, README) are at 62-75% coverage -- the uncovered lines are error return paths from `template.Execute` which are unreachable in normal testing since templates are embedded and validated at compile time.

The existing test suite already satisfies most of the bean requirements with structural assertions (per project testing rules). **The bean description recommends golden file testing, but the project testing rules in `.claude/rules/testing-patterns.md` explicitly require structural assertions over golden files.** The structural approach is used consistently across all existing tests and is the correct pattern for this codebase.

### Gaps Identified

1. ~~**Rendered extra domains in dynamic-domains.conf**~~ -- **DROPPED (challenge: already covered by `TestRenderFirewall_AllStacks` which passes extras through `Merge` and asserts each `cfg.Domains.Dynamic` entry appears in rendered `DynamicDomains`)**

2. **Python-specific domain rendering** -- The bean says "Render dynamic-domains.conf for Python has pypi.org". The firewall registry classifies `pypi.org` as Static (not Dynamic), so it would NOT appear in `dynamic-domains.conf`. However, Python has no Dynamic domains in the registry, so the correct test is that Python domains appear in `init-firewall.sh` (for static) and that the template handles stacks with no dynamic domains gracefully. No test currently verifies Python-specific domain rendering through the template pipeline.

3. **Systematic shebang/structure check for all shell scripts** -- Individual tests exist but there is no single test that checks all three shell scripts (init-firewall.sh, warmup-dns.sh, sync-claude-settings.sh) for shebangs and `set -euo pipefail` in one place.

4. **Dockerfile determinism** -- DevContainer and Claude have determinism tests, but Dockerfile does not.

5. **RenderFirewall determinism** -- No determinism test for RenderFirewall output.

6. **Dockerfile template artifact check** -- README and Claude tests check for `<no value>` / `<nil>` artifacts, but Dockerfile tests do not.

7. **All-stacks Dockerfile integration test** -- Dockerfile tests cover Go, Go+Node, Go+Ruby+Python, Ruby, but not all 5 stacks combined.

8. **DevContainer static template verification** -- The devcontainer.json template has no template actions (it is fully static), but no test proves this by rendering with different configs and asserting byte-equality. The JSON template testing rules say: "When a template has no Go template actions, render with different configs and assert byte-equality to prove it is truly stack-agnostic."

9. ~~**No `t.Parallel()` on any tests**~~ -- **DROPPED (challenge: tests run in <300ms, parallelizing adds churn without meaningful benefit; do separately if desired)**

### Approach

Add targeted tests to fill the 9 gaps above. No changes to source code. All new tests follow the structural assertion pattern established in the codebase.

### Files to Modify

- `internal/render/firewall_test.go` -- Add tests for Python domain rendering, all-scripts shebang/structure check, determinism
- `internal/render/dockerfile_test.go` -- Add determinism test, template artifact check, all-stacks integration test
- `internal/render/devcontainer_test.go` -- Add static template verification test

### Steps

1. ~~**Extra domains in rendered dynamic-domains.conf**~~ -- DROPPED (already covered)

2. **Python domain rendering through template pipeline** -- Add `TestRenderFirewall_PythonDomainsInInitFirewall` to `firewall_test.go`. Call `Merge([]stack.StackID{stack.Python}, nil)`, render, assert `pypi.org` and `files.pythonhosted.org` appear in `InitFirewall` (they are Static), and assert DynamicDomains has no Python-specific entries (Python has no dynamic domains in the registry).

3. **Systematic shell script shebang and structure test** -- Add `TestRenderFirewall_AllShellScripts_ShebangAndStrictMode` and `TestRenderClaude_SyncSettings_ShebangAndStrictMode` tests. For firewall: render with Go stack, check InitFirewall and WarmupDNS both have `#!/usr/bin/env bash` prefix and contain `set -euo pipefail`. For Claude: check SyncSettings has both. This consolidates the structural checks that exist individually into a systematic sweep.

4. **Dockerfile determinism** -- Add `TestDockerfile_Deterministic` to `dockerfile_test.go`. Merge Go+Node+Python, render twice, assert `out1 == out2`.

5. **RenderFirewall determinism** -- Add `TestRenderFirewall_Deterministic` to `firewall_test.go`. Merge Go+Node, render twice, assert all three files are byte-equal.

6. **Dockerfile template artifact check** -- Add `TestDockerfile_NoTemplateArtifacts` to `dockerfile_test.go`. Render with Go stack, check output does not contain `<no value>`, `<nil>`, `{{`, `}}`.

7. **All-stacks Dockerfile integration** -- Add `TestDockerfile_AllStacks` to `dockerfile_test.go`. Merge all 5 stacks, render, verify all runtimes and LSP install commands appear, verify system deps from Ruby+Python appear, verify node appears exactly once.

8. **DevContainer static template verification** -- Add `TestDevContainer_IsStatic` to `devcontainer_test.go`. Render with Go-only config and Go+Node+Python config, assert byte-equality. This proves the template has no dynamic actions.

9. ~~**Add `t.Parallel()` to all tests**~~ -- DROPPED (premature optimization)

### Testing Strategy

- All new tests use structural assertions, not golden files (per project rules)
- New tests follow the two-tier strategy: integration (through Merge + render) and isolation (hand-built GenerationConfig)
- Registry-computed completeness for domain checks (iterate cfg.Domains.* and assert each appears in output)
- Spot-checks for well-known entries (pypi.org, gopls, etc.)
- Verify the test suite still passes after changes: `go test ./internal/render/... -race -count=1`
- Verify lint passes: `golangci-lint run ./internal/render/...`

### Open Questions

1. **Golden files vs structural assertions** -- The bean description says "Use golden file testing pattern: expected outputs checked into testdata/." However, the project testing rules explicitly state "Use structural assertions, not golden-file snapshots." The existing 88 tests all use structural assertions. **Recommendation: Follow the project rules and use structural assertions.** Golden files would be fragile against registry changes (adding a domain breaks all golden files) and duplicate the template content rather than testing behavior.

## Checklist
- [x] Tests written
- [x] No TODO/FIXME/HACK/XXX comments
- [x] Lint passes
- [x] Tests pass
- [ ] Branch pushed
- [ ] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] All checklist items completed
- [ ] User notified

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | done | 1 | 2026-04-03 |
| challenge | done | 1 | 2026-04-03 |
| implement | done | 1 | 2026-04-03 |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |
