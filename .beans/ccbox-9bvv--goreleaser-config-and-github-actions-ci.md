---
# ccbox-9bvv
title: GoReleaser config and GitHub Actions CI
status: completed
type: task
priority: normal
created_at: 2026-04-02T10:33:59Z
updated_at: 2026-04-03T09:56:23Z
parent: ccbox-jxut
---

## Description
Set up GoReleaser config (`.goreleaser.yml`) for:
- Cross-platform builds: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- GitHub releases with changelog
- Homebrew tap formula generation (target: `bjro/homebrew-tap`)

Set up GitHub Actions:
- CI workflow: lint (golangci-lint), test, build on PRs
- Release workflow: triggered on tag push, runs GoReleaser

## Checklist
- [x] Tests written
- [x] No TODO/FIXME/HACK/XXX comments
- [x] Lint passes
- [x] Tests pass
- [ ] Branch pushed
- [ ] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] User notified

## Implementation Plan

### Approach

Create three config files: `.goreleaser.yml` for release builds, `.github/workflows/ci.yml` for PR checks (lint, test, build), and `.github/workflows/release.yml` for tag-triggered releases via GoReleaser. Since these are purely declarative config files (no Go source changes), testing strategy focuses on schema validation and dry-run verification rather than Go unit tests.

### Files to Create

- `.goreleaser.yml` — GoReleaser v2 configuration for cross-platform builds, ldflags version injection, and Homebrew tap formula generation
- `.github/workflows/ci.yml` — GitHub Actions CI workflow for PRs (lint, test, build)
- `.github/workflows/release.yml` — GitHub Actions release workflow triggered by version tags

### Steps

#### 1. Create `.goreleaser.yml`

GoReleaser v2 config with these sections:

**version**: Set to `2` (GoReleaser v2 config format).

**builds**: Single build entry for the `ccbox` binary.
- `main: .` (main.go is at repo root)
- `binary: ccbox`
- `env: [CGO_ENABLED=0]` for static binaries
- `goos: [linux, darwin]`
- `goarch: [amd64, arm64]`
- `ldflags: -s -w -X github.com/bjro/ccbox/cmd.version={{.Version}}`
  - `-s -w` strips debug info for smaller binaries
  - `-X ...` injects the version from the Git tag, matching the existing `var version = "dev"` in `cmd/root.go`

**archives**: Single archive entry.
- `formats: [tar.gz]` (standard for Go CLI tools)
- `name_template: ccbox_{{ .Version }}_{{ .Os }}_{{ .Arch }}`

**checksum**: Enable SHA256 checksums file.
- `name_template: checksums.txt`

**changelog**: Auto-generate from Git commits.
- `sort: asc`
- `filters.exclude`: Filter out docs, chore, ci commit prefixes to keep the changelog focused on user-facing changes

**brews**: Homebrew tap formula generation.
- `repository.owner: bjro`
- `repository.name: homebrew-tap`
- `repository.token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"` (dedicated PAT for cross-repo push)
- `directory: Formula` (standard Homebrew convention)
- `homepage: https://github.com/BjRo/ccbox`
- `description: Generate devcontainer setups for Claude Code`
- `license: MIT`
- `install`: Standard `bin.install "ccbox"` formula
- `test`: `system "#{bin}/ccbox", "--version"` to verify the formula works

Note: The `brews` section requires a `HOMEBREW_TAP_TOKEN` GitHub token with write access to the `bjro/homebrew-tap` repository. This must be configured as a repository secret.

#### 2. Create `.github/workflows/ci.yml`

CI workflow triggered on pull requests and pushes to `main`.

**name**: `CI`

**on**:
- `push.branches: [main]` — run on direct pushes to main
- `pull_request` — run on all PRs (no branch filter needed)

**permissions**: `contents: read` (principle of least privilege)

**jobs**:

**lint** job:
- `runs-on: ubuntu-latest`
- Steps:
  1. `actions/checkout@v4`
  2. `actions/setup-go@v5` with `go-version-file: go.mod` (reads version from go.mod so we never need to update the workflow when Go version changes)
  3. `golangci/golangci-lint-action@v7` with `version: v2` (tracks latest v2.x release; using v7 of the action which supports golangci-lint v2)

**test** job:
- `runs-on: ubuntu-latest`
- Steps:
  1. `actions/checkout@v4`
  2. `actions/setup-go@v5` with `go-version-file: go.mod`
  3. `go test ./...` — unit tests only (integration tests require `-tags integration` and may need a more complex setup; keep CI fast)

**build** job:
- `runs-on: ubuntu-latest`
- Steps:
  1. `actions/checkout@v4`
  2. `actions/setup-go@v5` with `go-version-file: go.mod`
  3. `go build ./...` — verify the project compiles

All three jobs run in parallel (no `needs` dependencies between them) for maximum CI speed.

#### 3. Create `.github/workflows/release.yml`

Release workflow triggered on version tag pushes.

**name**: `Release`

**on**:
- `push.tags: ["v*"]` — trigger on any `v`-prefixed tag (e.g., `v0.1.0`, `v1.0.0-rc.1`)

**permissions**:
- `contents: write` (GoReleaser needs to create GitHub releases and upload assets)

**jobs**:

**release** job:
- `runs-on: ubuntu-latest`
- Steps:
  1. `actions/checkout@v4` with `fetch-depth: 0` (GoReleaser needs full Git history for changelog generation)
  2. `actions/setup-go@v5` with `go-version-file: go.mod`
  3. `goreleaser/goreleaser-action@v6` with:
     - `distribution: goreleaser` (OSS edition)
     - `version: "~> v2"` (match GoReleaser v2 config format)
     - `args: release --clean`
  4. `env.GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}` — for creating releases
  5. `env.HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}` — for pushing to the Homebrew tap repo

#### 4. Update `.gitignore`

Add `dist/` to `.gitignore`. GoReleaser creates a `dist/` directory for build artifacts during local runs and the `--clean` flag removes it before each run, but we should never commit it.

### Testing Strategy

Since the deliverables are declarative config files (YAML), not Go source code, the standard Go unit test approach does not apply. Instead:

1. **GoReleaser schema validation**: Run `goreleaser check` locally to verify the `.goreleaser.yml` config is valid.
2. **GitHub Actions syntax validation**: GitHub validates workflow YAML on push. Syntax errors surface immediately as failed workflow runs.
3. **Smoke test after first tag**: The sibling bean `ccbox-xeg2` covers the end-to-end release verification.
4. **CI workflow verification**: After the PR is created, the CI workflow will run against the PR itself, providing immediate validation.
5. **No Go test changes needed**: No Go source files are modified, so existing tests remain valid.

### Design Decisions

- **Three jobs in CI, not one**: Lint, test, and build run as separate parallel jobs for faster feedback.
- **`go-version-file: go.mod`**: Single-source-of-truth for Go version.
- **No integration tests in CI**: Keep `go test ./...` fast for TDD loops.
- **GoReleaser v2 config format**: Matches the project setup.
- **Separate Homebrew tap token**: The `GITHUB_TOKEN` only has permissions for the current repository.
- **`CGO_ENABLED=0`**: Static binaries for a pure-Go CLI.
- **Tag pattern `v*`**: Matches Go module versioning conventions.

### Open Questions

None.

## Challenge Report

**Scope: SMALL CHANGE** (4 files: `.goreleaser.yml`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`, `.gitignore` update)

### Scope Assessment

| Metric | Value | Threshold |
|--------|-------|-----------|
| Files | 4 (3 new, 1 modified) | >15 = recommend split |

### Findings

#### Go Engineer

> **Finding 1: `repository.token` missing from Step 1 brews specification** (severity: WARNING)
>
> Step 1 lists the `.goreleaser.yml` `brews` section fields (lines 74-82) but omits `repository.token`. The token reference (`repository.token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"`) only appears in a prose note at the bottom of Step 3 (line 149). An implementer following Step 1's bullet-point structure literally will produce a `.goreleaser.yml` that creates the GitHub release successfully but silently fails to push the Homebrew formula -- GoReleaser will skip the brew step without a token rather than erroring. This is the kind of "works on first glance, fails in production" gap that causes hours of debugging.
>
> **Option A (recommended):** Add `repository.token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"` as an explicit bullet under the `brews` field list in Step 1, right after `repository.name`. Remove the ambiguous note on line 84 that says "pass it via `GITHUB_TOKEN` or a dedicated PAT" since the decision is clearly a dedicated PAT.
> **Option B:** Keep the current structure but add a cross-reference from Step 1 to Step 3's note (e.g., "See Step 3 for token configuration").

> **Finding 2: `golangci-lint` version `v2.11` may not exist and pinning strategy is fragile** (severity: SUGGESTION)
>
> Step 2 pins golangci-lint to `version: v2.11`. As of the plan date, the latest golangci-lint v2 release may be at a different minor version. The golangci-lint-action@v7 `version` field expects an exact version or `latest`. Pinning to a non-existent version will cause the lint job to fail on the very first CI run -- a bad first impression for a self-validating deliverable. More practically, pinning to an exact minor version means the workflow needs manual updates for golangci-lint patches, which adds maintenance burden with no safety benefit (the `.golangci.yml` config format is stable across v2 minors).
>
> **Suggestion:** Use `version: "v2"` to track the latest v2.x release, or verify the exact latest version at implementation time and document why it was chosen.

### Verdict

**APPROVED**

The plan is solid and well-structured for its scope. The GoReleaser v2 syntax (`version: 2`, `formats`, `brews` with `repository` sub-fields), GitHub Actions patterns (`fetch-depth: 0`, `go-version-file`, separate permissions), and the overall architecture (parallel CI jobs, tag-triggered release, separate Homebrew tap token) are all correct. The `ldflags` path matches the existing `var version = "dev"` in `cmd/root.go`. The decision to use `CGO_ENABLED=0` is appropriate for a pure-Go CLI. The testing strategy is pragmatic -- these are declarative config files, and the CI workflow is self-validating on the first PR.

Finding 1 is the only one that matters for implementation correctness. The `repository.token` field must appear in the `.goreleaser.yml` or the Homebrew tap push will silently not happen. This is fixable during implementation without plan revision.

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-03 |
| challenge | complete | 1 | 2026-04-03 |
| implement | complete | 1 | 2026-04-03 |
| pr | blocked | | 2026-04-03 |
| review | pending | | |
| codify | pending | | |

## Agent Checkpoint

Implementation complete. All 4 files created/updated:
1. `.goreleaser.yml` - with repository.token per Finding 1
2. `.github/workflows/ci.yml` - with golangci-lint version: "v2" per Finding 2
3. `.github/workflows/release.yml` - with HOMEBREW_TAP_TOKEN env
4. `.gitignore` - added dist/

Lint passes, tests pass. Committed as d6425ea.

Push blocked: The current OAuth token (gho_ prefix) lacks the `workflow` scope required
to push `.github/workflows/` files. This requires either:
- A PAT with `workflow` scope, or
- SSH key access configured in this environment

The user needs to push manually with appropriate credentials.