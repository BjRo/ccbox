---
# ccbox-ff1i
title: Domain merging and deduplication
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:35:46Z
updated_at: 2026-04-02T12:51:55Z
parent: ccbox-m6ll
---

## Description
Implement the logic that produces the final domain lists from:
1. Always-on domains (GitHub, Anthropic, etc.)
2. Per-stack default domains (from stack metadata registry)
3. User-specified extra domains (from wizard or CLI flags)

Output two lists:
- **Static domains**: For init-firewall.sh to resolve once and add to ipset
- **Dynamic domains**: For dynamic-domains.conf / dnsmasq

Dedup across all sources. User extras go into dynamic domains by default (safer — handles IP changes).

## Checklist

- [ ] Tests written (TDD)
- [ ] No TODO/FIXME/HACK/XXX in new code
- [ ] Lint passes (`golangci-lint run ./...`)
- [ ] Tests pass (`go test ./...`)
- [ ] Branch pushed
- [ ] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] ADR written (if architectural changes)

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | pending | | |
| challenge | pending | | |
| implement | pending | | |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |