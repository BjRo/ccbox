---
# ccbox-ztaa
title: Curate per-stack domain allowlists
status: todo
type: task
priority: high
created_at: 2026-04-02T10:35:44Z
updated_at: 2026-04-02T10:35:44Z
parent: ccbox-m6ll
---

## Description
Research and curate the default domain allowlists for each supported stack. Domains fall into two categories:

**Static domains** (resolved once at firewall init, IPs cached in ipset):
- Domains with stable IPs (e.g., `api.github.com` — fetched as CIDRs from GitHub meta API)

**Dynamic domains** (managed by dnsmasq, re-resolved periodically):
- CDNs and services with rotating IPs

**Per-stack lists to curate:**

| Stack | Static | Dynamic |
|-------|--------|---------|
| Always-on | api.github.com, *.anthropic.com, sentry.io, statsig | — |
| Go | — | proxy.golang.org, sum.golang.org, storage.googleapis.com |
| Node | registry.npmjs.org | cdn.jsdelivr.net, unpkg.com |
| Python | pypi.org, files.pythonhosted.org | — |
| Rust | crates.io, static.crates.io | — |
| Ruby | rubygems.org, index.rubygems.org | — |

Validate each domain is actually needed for basic development workflows (install deps, run tests, use Claude Code). Document the rationale.