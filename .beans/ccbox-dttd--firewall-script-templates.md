---
# ccbox-dttd
title: Firewall script templates
status: in-progress
type: task
priority: high
created_at: 2026-04-02T10:35:24Z
updated_at: 2026-04-02T15:27:54Z
parent: ccbox-6z26
---

## Description
Three templates based on credfolio2 reference:

**init-firewall.sh** (parameterized):
- Core structure same as reference: preserve Docker DNS, flush rules, create ipset, default DROP policy
- Static domain resolution: always-on domains (GitHub API for CIDRs, Anthropic, npmjs, etc.) + stack-specific static domains
- dnsmasq setup for dynamic domains
- Verification tests at the end
- Template variable: list of static domains, list of dynamic domains

**warmup-dns.sh** (static):
- Reads from dynamic-domains.conf, resolves each via dig through dnsmasq
- No parameterization needed, works off dynamic-domains.conf

**dynamic-domains.conf** (parameterized):
- Stack-specific dynamic domains (e.g., `proxy.golang.org` for Go, `cdn.jsdelivr.net` for Node)
- User-specified extra domains appended
- One domain per line, comments for sections

## Checklist

- [ ] Tests written and passing (TDD)
- [ ] No TODO/FIXME/HACK/XXX comments in new code
- [ ] Lint passes (`golangci-lint run ./...`)
- [ ] All tests pass (`go test ./...`)
- [ ] Branch pushed
- [ ] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] ADR written (if architectural changes)
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