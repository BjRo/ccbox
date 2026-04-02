---
# ccbox-dttd
title: Firewall script templates
status: todo
type: task
priority: high
created_at: 2026-04-02T10:35:24Z
updated_at: 2026-04-02T10:35:24Z
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