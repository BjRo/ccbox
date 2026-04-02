---
# ccbox-3wvl
title: Unit tests for template generation
status: todo
type: task
priority: normal
created_at: 2026-04-02T10:36:00Z
updated_at: 2026-04-02T10:36:00Z
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