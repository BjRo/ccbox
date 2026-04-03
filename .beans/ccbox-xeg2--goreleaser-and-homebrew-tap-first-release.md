---
# ccbox-xeg2
title: GoReleaser and Homebrew tap first release
status: completed
type: task
priority: normal
created_at: 2026-04-02T10:36:23Z
updated_at: 2026-04-03T10:05:27Z
parent: ccbox-vydo
---

## Description
Cut the first release (v0.1.0):

1. Create `bjro/homebrew-tap` repo on GitHub (or reuse if exists)
2. Configure GoReleaser to publish formula to the tap
3. Tag v0.1.0 and push
4. Verify: `brew install bjro/tap/ccbox` works
5. Verify: GitHub release has binaries for linux/darwin × amd64/arm64
6. Announce on README with installation instructions