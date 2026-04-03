---
# ccbox-9bvv
title: GoReleaser config and GitHub Actions CI
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:33:59Z
updated_at: 2026-04-03T09:16:54Z
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

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | pending | | |
| challenge | pending | | |
| implement | pending | | |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |