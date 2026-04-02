---
# ccbox-txxy
title: 'Integration tests: end-to-end generation'
status: todo
type: task
priority: normal
created_at: 2026-04-02T10:36:03Z
updated_at: 2026-04-02T10:36:03Z
parent: ccbox-6g75
---

## Description
End-to-end tests that run `ccbox init` against sample project directories and validate the full output:

1. Create temp dir with go.mod → run ccbox init --non-interactive → verify all 8 files exist with correct content
2. Create temp dir with package.json + go.mod → run ccbox init --non-interactive → verify multi-stack output
3. Create temp dir with existing .devcontainer/ → run ccbox init → verify it aborts with error message
4. Create temp dir with go.mod → run ccbox init --extra-domains "api.example.com" → verify domain appears in dynamic-domains.conf
5. Verify .ccbox.yml is written with correct stacks and options

Run as `go test -tags=integration` to separate from unit tests.