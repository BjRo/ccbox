---
# agentbox-ee92
title: 'Fix firewall script: resolv.conf rewrite, upstream DNS config, hash:net ipset'
status: todo
type: bug
priority: critical
created_at: 2026-04-07T16:54:45Z
updated_at: 2026-04-07T16:55:05Z
parent: agentbox-el52
---

The generated init-firewall.sh has critical bugs causing DNS-based firewall rules to go stale. See bean body for details.

## Root Cause

The firewall template was based on a working reference script (credfolio2) but the implementation missed critical operational plumbing:

1. **`/etc/resolv.conf` never updated to use dnsmasq** — dnsmasq runs but nothing uses it. Normal DNS queries from git, curl, go etc. go through Docker DNS directly, bypassing dnsmasq's ipset hooks. The ipset only gets populated during warmup (which uses `@127.0.0.1` explicitly), then goes stale as IPs rotate.

2. **No upstream DNS forwarding config for dnsmasq** — The reference script captures Docker's upstream DNS, caches it for restarts, and configures dnsmasq with `no-resolv` + explicit `server=` directives. Without this, pointing resolv.conf at 127.0.0.1 creates a circular dependency.

3. **`hash:ip` instead of `hash:net`** — ipset created with `hash:ip` which doesn't support CIDR ranges. Should be `hash:net` to support GitHub's published CIDR ranges.

4. **No GitHub meta API CIDR fetch** — The reference script fetches `api.github.com/meta` to get GitHub's full IP ranges. Current script relies solely on single `dig` lookups.

5. **No Docker NAT rule preservation** — The reference script saves/restores Docker DNS NAT rules across iptables operations.

## Fix

Port the reference script's approach into init-firewall.sh.tmpl:

- Capture upstream DNS before any changes, cache for restarts
- Configure dnsmasq with `no-resolv` + explicit `server=` lines
- Rewrite `/etc/resolv.conf` to point at `127.0.0.1` after dnsmasq starts
- Use `hash:net` for ipset
- Fetch GitHub meta API CIDRs
- Preserve Docker DNS NAT rules across iptables flushes
- Add proper verification (blocked + allowed domain tests)

The dynamic-domains.conf.tmpl and warmup-dns.sh.tmpl are fine as-is.

## Definition of Done

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [x] Branch pushed to remote
- [ ] PR created
- [ ] Automated code review passed via `@review-backend` subagent (via Task tool)
- [ ] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review

## Implementation Plan

### Approach

Rewrite the `init-firewall.sh.tmpl` template to fix all five root-cause bugs. The fix is entirely within the template layer -- no changes to Go types, the `GenerationConfig` struct, or the `firewall.Merge` pipeline. The template data model (`{{range .Domains.Static}}`, `{{range .Domains.Dynamic}}`, `{{stripWildcard .Name}}`) is preserved exactly as-is.

The new script follows the reference script's proven architecture: capture upstream DNS early, cache it for idempotent restarts, save/restore Docker NAT rules across iptables flushes, configure dnsmasq with explicit upstream servers, rewrite `/etc/resolv.conf`, and add proper verification.

Additionally, the README template's mention of `hash:ip` must be updated to `hash:net`.

### Files to Create/Modify

- `internal/render/templates/init-firewall.sh.tmpl` -- Complete rewrite of the firewall init script template (the core bug fix)
- `internal/render/templates/README.md.tmpl` -- Change `hash:ip` to `hash:net` in the firewall architecture section (line 38)
- `internal/render/firewall_test.go` -- Update and add tests to verify the new script structure

### Steps

#### 1. Update tests first (TDD)

Modify `/workspace/internal/render/firewall_test.go`:

**a. Update `TestRenderFirewall_InitFirewall_ScriptStructure`**
- Add structural markers for the new script sections:
  - `UPSTREAM_DNS_CACHE` (upstream DNS caching file path)
  - `no-resolv` (dnsmasq config directive)
  - `server=` (upstream DNS forwarding in dnsmasq config)
  - `nameserver 127.0.0.1` (resolv.conf rewrite)
  - `hash:net` (ipset type, replacing `hash:ip`)
  - `api.github.com/meta` (GitHub CIDR fetch)
  - `iptables-save` and `iptables-restore` (NAT rule preservation)
  - `example.com` (negative verification test domain)
- Remove `hash:ip` from expected markers (negative check)

**b. Add `TestRenderFirewall_InitFirewall_HashNetNotHashIP`**
- Assert `hash:net` is present in rendered output
- Assert `hash:ip` is NOT present in rendered output
- This explicitly guards against regression to the old ipset type

**c. Add `TestRenderFirewall_InitFirewall_ResolvConfRewrite`**
- Assert output contains `nameserver 127.0.0.1`
- Assert output contains `resolv.conf` (the rewrite target)
- Assert output contains `no-resolv` (dnsmasq directive preventing resolv.conf loop)

**d. Add `TestRenderFirewall_InitFirewall_UpstreamDNS`**
- Assert output contains `UPSTREAM_DNS_CACHE` or equivalent cache file reference
- Assert output contains `server=` directive for dnsmasq upstream forwarding

**e. Add `TestRenderFirewall_InitFirewall_NATPreservation`**
- Assert output contains `iptables-save` and `iptables-restore`
- These are the specific commands used to save/restore Docker DNS NAT rules

**f. Add `TestRenderFirewall_InitFirewall_Verification`**
- Assert output contains both a positive test (`github.com` or similar allowed domain)
- Assert output contains a negative test (`example.com` for blocked domain verification)

**g. Add `TestRenderFirewall_InitFirewall_SSHOutbound`**
- Assert output contains port 22 in an iptables ACCEPT rule context

**h. Add `TestRenderFirewall_InitFirewall_HostNetwork`**
- Assert output contains `172.16.0.0/12` or Docker host network range in iptables rules

**i. Update `TestRenderFirewall_AllStacks`**
- Verify the `hash:net` marker is present (structural completeness check)

**j. Add `TestRenderFirewall_InitFirewall_NoTemplateArtifacts`**
- Render with all 5 stacks to exercise every template branch
- Assert no `<no value>`, `<nil>`, `{{`, or `}}` artifacts
- This is the all-stacks artifact check required by testing patterns

#### 2. Rewrite `init-firewall.sh.tmpl`

Replace the entire template at `/workspace/internal/render/templates/init-firewall.sh.tmpl` with a new script that follows this section structure. All existing Go template actions (`{{range .Domains.Static}}`, `{{range .Domains.Dynamic}}`, `{{stripWildcard .Name}}`) are preserved in their same functional roles.

**Section 1: Upstream DNS capture and caching**
```
UPSTREAM_DNS_CACHE="/var/cache/agentbox-upstream-dns"
```
- On first run: extract Docker DNS from `/etc/resolv.conf`, write to cache file
- On subsequent runs (container restart): read from cache file
- This makes the script idempotent -- once resolv.conf is rewritten to 127.0.0.1, the original Docker DNS is still available from cache

**Section 2: Save Docker DNS NAT rules**
- Use `iptables-save -t nat` to capture Docker's NAT rules before any iptables flush
- Store in a temp file using the `trap`/`EXIT` cleanup pattern per project rules

**Section 3: Flush iptables rules cleanly**
- `iptables -F` (flush filter chains)
- `iptables -X` (delete user chains)
- `iptables -t nat -F` and `-X` for NAT table
- Restore saved Docker NAT rules via `iptables-restore --noflush`

**Section 4: Create ipset with `hash:net`**
- `ipset create allowed_ips hash:net -exist`
- `ipset flush allowed_ips` for idempotent re-runs

**Section 5: Resolve static domains**
- Same `{{range .Domains.Static}}` loop as current template
- Same `dig +short` with IPv4 regex filter and single-quoted domain names
- Same `ipset add allowed_ips "${ip}" -exist` pattern
- Add CIDR support: after the dig loop, check if the domain is `api.github.com` and if so, fetch `https://api.github.com/meta` via curl and extract CIDR ranges from the `git`, `web`, `api`, `actions` arrays using jq, adding each to the ipset

**Section 6: Install and configure dnsmasq**
- Write dnsmasq config to `/etc/dnsmasq.d/agentbox.conf` (renamed from `agentbox-dynamic.conf` to reflect it now contains both ipset directives AND upstream DNS config):
  - `no-resolv` directive (prevents dnsmasq from reading resolv.conf, breaking the circular dependency)
  - `server=${UPSTREAM_DNS}` directive (explicit upstream forwarding)
  - Same `{{range .Domains.Dynamic}}` ipset directives as current template
- Restart dnsmasq (same `systemctl`/`service` fallback as current)

**Section 7: Rewrite `/etc/resolv.conf`**
- `echo "nameserver 127.0.0.1" > /etc/resolv.conf`
- This is the critical missing step: all DNS queries now go through dnsmasq, which populates the ipset via its ipset hooks

**Section 8: Base iptables rules**
- Allow loopback: `iptables -A OUTPUT -o lo -j ACCEPT`
- Allow established: `iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT`
- Allow Docker DNS (upstream): `iptables -A OUTPUT -d "${UPSTREAM_DNS}" -p udp --dport 53 -j ACCEPT` and same for tcp
- Allow SSH outbound (port 22): `iptables -A OUTPUT -p tcp --dport 22 -j ACCEPT`
- Allow Docker host network: `iptables -A OUTPUT -d 172.16.0.0/12 -j ACCEPT`
- Allow ipset: `iptables -A OUTPUT -m set --match-set allowed_ips dst -j ACCEPT`
- Default DROP: `iptables -P OUTPUT DROP`

**Section 9: Run DNS warmup**
- Same `bash "${SCRIPT_DIR}/warmup-dns.sh"` as current template

**Section 10: Verification**
- Positive test: `curl -sf --max-time 5 https://api.github.com/zen > /dev/null 2>&1` (verifies full HTTPS connectivity to an allowed domain, not just DNS resolution)
- Negative test: `curl -sf --max-time 5 https://example.com > /dev/null 2>&1` should fail (verifies that non-allowed domains are actually blocked)
- Log pass/fail for both tests

**Key template action preservation:**
- `{{- range .Domains.Static}}` with `'{{.Name}}'` and `{{.Rationale}}` -- same position (Section 5)
- `{{- range .Domains.Dynamic}}` with `{{stripWildcard .Name}}` -- same position (Section 6, inside dnsmasq config heredoc)
- All domain names remain single-quoted in shell contexts per project rules

#### 3. Update `README.md.tmpl`

In `/workspace/internal/render/templates/README.md.tmpl`, line 38:
- Change `hash:ip` to `hash:net` in the firewall architecture description

#### 4. Verify all tests pass

Run the full test suite:
- `go test ./internal/render/...` (unit tests for the changed package)
- `go test ./...` (full unit test suite)
- `golangci-lint run ./...` (linting)

### Testing Strategy

**Tests to write** (all in `/workspace/internal/render/firewall_test.go`):

1. **`TestRenderFirewall_InitFirewall_HashNetNotHashIP`** -- Assert `hash:net` present, `hash:ip` absent. Guards against the ipset type regression.

2. **`TestRenderFirewall_InitFirewall_ResolvConfRewrite`** -- Assert `nameserver 127.0.0.1`, `resolv.conf`, and `no-resolv` are present. Validates the core bug fix.

3. **`TestRenderFirewall_InitFirewall_UpstreamDNS`** -- Assert upstream DNS caching and `server=` directive. Validates dnsmasq won't create a DNS loop.

4. **`TestRenderFirewall_InitFirewall_NATPreservation`** -- Assert `iptables-save` and `iptables-restore` present. Validates Docker DNS NAT rules survive iptables flush.

5. **`TestRenderFirewall_InitFirewall_Verification`** -- Assert both positive (allowed domain) and negative (`example.com`) test domains appear in verification section.

6. **`TestRenderFirewall_InitFirewall_SSHOutbound`** -- Assert port 22 ACCEPT rule.

7. **`TestRenderFirewall_InitFirewall_HostNetwork`** -- Assert Docker host network range in iptables rules.

8. **`TestRenderFirewall_InitFirewall_NoTemplateArtifacts`** -- All-stacks render with artifact checks for `<no value>`, `<nil>`, `{{`, `}}`.

9. **Update `TestRenderFirewall_InitFirewall_ScriptStructure`** -- Add new structural markers, remove old `hash:ip` expectation.

**Tests to verify still pass** (existing, no changes needed):
- `TestRenderFirewall_NoError` -- Still renders without error
- `TestRenderFirewall_InitFirewall_ContainsStaticDomains` -- Static domains still present
- `TestRenderFirewall_InitFirewall_ContainsDynamicDomains` -- Dynamic dnsmasq directives still present
- `TestRenderFirewall_EmptyDomains` -- Empty input still produces valid output
- `TestRenderFirewall_InitFirewall_SingleQuotedDomains` -- Defense-in-depth quoting preserved
- `TestRenderFirewall_Deterministic` -- Two renders produce identical output
- `TestRenderFirewall_AllStacks` -- All stacks produce valid output
- `TestRenderFirewall_WildcardDomainHandling` -- Wildcard stripping preserved
- All warmup-dns.sh and dynamic-domains.conf tests unchanged

### Open Questions

None -- all ambiguities resolved by examining the codebase and the reference script requirements. The data model (`GenerationConfig`, `MergedDomains`) fully supports all changes without modification.
