---
# agentbox-ibfc
title: 'Fix firewall script: resolv.conf rewrite, upstream DNS config, hash:net ipset'
status: todo
type: bug
priority: critical
created_at: 2026-04-07T16:54:41Z
updated_at: 2026-04-07T16:54:41Z
parent: agentbox-el52
---

The generated init-firewall.sh has several critical bugs that cause DNS-based firewall rules to go stale, requiring manual re-runs of init-firewall.sh when IPs rotate.

## Root Cause

The firewall template was based on a working reference script (credfolio2) but the implementation missed critical operational plumbing:

1. **`/etc/resolv.conf` never updated to use dnsmasq** — dnsmasq runs but nothing uses it. Normal DNS queries from git, curl, go etc. go through Docker DNS directly, bypassing dnsmasq's ipset hooks. The ipset only gets populated during warmup (which uses `@127.0.0.1` explicitly), then goes stale as IPs rotate.

2. **No upstream DNS forwarding config for dnsmasq** — The reference script captures Docker's upstream DNS, caches it for restarts, and configures dnsmasq with `no-resolv` + explicit `server=` directives. Without this, pointing resolv.conf at 127.0.0.1 creates a circular dependency.

3. **`hash:ip` instead of `hash:net`** — ipset created with `hash:ip` which doesn't support CIDR ranges. Should be `hash:net` to support GitHub's published CIDR ranges.

4. **No GitHub meta API CIDR fetch** — The reference script fetches `api.github.com/meta` to get GitHub's full IP ranges and aggregates them into the ipset. Current script relies solely on single `dig` lookups which miss IPs that GitHub load-balances across.

5. **No Docker NAT rule preservation** — The reference script saves/restores Docker DNS NAT rules across iptables operations. Current script doesn't handle this.

## Fix

Port the reference script's approach into the init-firewall.sh.tmpl template:

- Capture upstream DNS before any changes, cache for restarts
- Configure dnsmasq with `no-resolv` + explicit `server=` lines
- Rewrite `/etc/resolv.conf` to point at `127.0.0.1` after dnsmasq starts
- Use `hash:net` for ipset
- Fetch GitHub meta API CIDRs
- Preserve Docker DNS NAT rules across iptables flushes
- Add proper verification (blocked domain test + allowed domain test)

The dynamic-domains.conf.tmpl and warmup-dns.sh.tmpl are fine as-is. The fix is entirely in init-firewall.sh.tmpl.
