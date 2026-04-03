---
# ccbox-9bvv
title: GoReleaser config and GitHub Actions CI
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:33:59Z
updated_at: 2026-04-03T09:18:48Z
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
- [ ] Tests written
- [ ] No TODO/FIXME/HACK/XXX comments
- [ ] Lint passes
- [ ] Tests pass
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
- `directory: Formula` (standard Homebrew convention)
- `homepage: https://github.com/BjRo/ccbox`
- `description: Generate devcontainer setups for Claude Code`
- `license: MIT`
- `install`: Standard `bin.install "ccbox"` formula
- `test`: `system "#{bin}/ccbox", "--version"` to verify the formula works

Note: The `brews` section requires a `HOMEBREW_TAP_TOKEN` GitHub token with write access to the `bjro/homebrew-tap` repository. This must be configured as a repository secret. The release workflow will pass it via `GITHUB_TOKEN` or a dedicated PAT.

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
  3. `golangci/golangci-lint-action@v7` with `version: v2.11` (pin to minor version to match the v2 config format in `.golangci.yml`; using v7 of the action which supports golangci-lint v2)

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
  5. `env.HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}` — for pushing to the Homebrew tap repo (GoReleaser uses this to authenticate when pushing the formula to `bjro/homebrew-tap`)

Note on Homebrew tap token: GoReleaser needs a PAT (Personal Access Token) or fine-grained token with `contents: write` permission on the `bjro/homebrew-tap` repository. The default `GITHUB_TOKEN` only has permissions for the current repo. This token must be added as a repository secret named `HOMEBREW_TAP_TOKEN`. In the `.goreleaser.yml`, the brews section references this via `repository.token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"`.

#### 4. Update `.gitignore`

Add `dist/` to `.gitignore`. GoReleaser creates a `dist/` directory for build artifacts during local runs and the `--clean` flag removes it before each run, but we should never commit it.

### Testing Strategy

Since the deliverables are declarative config files (YAML), not Go source code, the standard Go unit test approach does not apply. Instead:

1. **GoReleaser schema validation**: Run `goreleaser check` (or `goreleaser build --snapshot --clean` for a dry-run build) in the CI pipeline or locally to verify the `.goreleaser.yml` config is valid. This is a manual verification step during implementation, not an automated test.

2. **GitHub Actions syntax validation**: GitHub validates workflow YAML on push. Syntax errors surface immediately as failed workflow runs. For local pre-validation, `actionlint` can be used if available.

3. **Smoke test after first tag**: The sibling bean `ccbox-xeg2` covers the end-to-end release verification (tag v0.1.0, verify binaries, verify Homebrew install). This bean only sets up the config; the first actual release is out of scope.

4. **CI workflow verification**: After the PR is created, the CI workflow will run against the PR itself, providing immediate validation that lint/test/build jobs work correctly. This is a self-validating deliverable.

5. **No Go test changes needed**: No Go source files are modified, so existing tests remain valid. The `go test ./...` and `golangci-lint run ./...` commands in CI exercise the exact same commands documented in CLAUDE.md.

### Design Decisions

- **Three jobs in CI, not one**: Lint, test, and build run as separate parallel jobs. This gives faster feedback (a lint failure shows immediately without waiting for tests) and clearer status checks on PRs.

- **`go-version-file: go.mod`**: Instead of hardcoding `go-version: "1.25"`, we read from `go.mod`. This is the single-source-of-truth approach — when Go version bumps, only `go.mod` changes.

- **No integration tests in CI**: The CI workflow runs `go test ./...` (unit tests only). Integration tests (`-tags integration`) are excluded because they exercise real filesystem operations that may need specific setup. They can be added to a separate CI job later if needed.

- **GoReleaser v2 config format**: Matches the project setup. Key difference from v1: the `version: 2` field at the top of `.goreleaser.yml` and updated schema for `brews` (was `brews` in v1 too, but field names differ slightly in v2).

- **Separate Homebrew tap token**: The `GITHUB_TOKEN` provided by GitHub Actions only has permissions for the current repository. Writing to `bjro/homebrew-tap` requires a separate PAT stored as `HOMEBREW_TAP_TOKEN` secret.

- **`CGO_ENABLED=0`**: ccbox is a pure Go project with no C dependencies. Static binaries simplify distribution and avoid glibc version mismatches across Linux distributions.

- **Tag pattern `v*`**: Matches Go module versioning conventions (`v0.1.0`, `v1.0.0`). The release workflow ignores non-version tags.

### Open Questions

None. The scope is well-defined by the bean description and existing project conventions. The Homebrew tap repository (`bjro/homebrew-tap`) and its PAT secret are prerequisites for the release workflow but are out of scope for this bean (covered by `ccbox-xeg2`).

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-03 |
| challenge | pending | | |
| implement | pending | | |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |