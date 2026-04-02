---
# ccbox-dttd
title: Firewall script templates
status: in-progress
type: task
priority: high
created_at: 2026-04-02T10:35:24Z
updated_at: 2026-04-02T15:36:12Z
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


## Implementation Plan

### Approach

Add three embedded Go templates to `internal/render/` and a rendering function that executes them against `GenerationConfig`. This is the FIRST template rendering code in the codebase, so it also establishes the embed+template pattern that all subsequent template beans (ccbox-v1zh, ccbox-v9jt, ccbox-780o, ccbox-7qvl) will follow.

The templates produce shell scripts and a config file for Linux container network isolation using iptables, ipset, and dnsmasq. Static domains (stable IPs) are resolved once at container init and loaded into an ipset. Dynamic domains (CDN/rotating IPs) are managed by dnsmasq with ipset hooks for automatic re-resolution.

Key design decisions:
- **`text/template` not `html/template`**: These are shell scripts, not HTML. No HTML escaping needed. However, `text/template` performs NO escaping at all, so defense-in-depth via input validation is required (see domain validation below).
- **Domain name validation (defense-in-depth)**: User-supplied domain names from `userExtraDomains` are interpolated into `init-firewall.sh` which runs as root. A malicious input like `; rm -rf /` or `$(cmd)` would be injected into shell commands. Add a `ValidateDomain` function in the `firewall` package that rejects inputs not matching a strict RFC 1123 DNS hostname pattern: alphanumeric characters, hyphens, dots, with an optional leading `*.` for wildcards. This validation runs in `firewall.Merge` before any domain is accepted. Combined with single-quoting domain names in all shell template interpolation points, this provides two independent layers of protection.
- **Templates embedded via `//go:embed`**: Bundle `.tmpl` files into the binary at compile time. Templates live in `internal/render/templates/` subdirectory to keep them separate from Go source.
- **Single `RenderFirewall` function**: Takes a `GenerationConfig` and returns rendered content as a `FirewallFiles` struct (filename-to-bytes), deferring actual file writing to a later bean (the orchestrator that calls `ccbox init`). This keeps `render` pure and testable without filesystem I/O.
- **`stripWildcard` template FuncMap helper**: The firewall registry contains `*.anthropic.com` as a Dynamic domain. The `*.` prefix must be stripped for two contexts: (1) `dynamic-domains.conf` outputs bare domain names for `dig` resolution (dig cannot resolve `*.anthropic.com`), and (2) `init-firewall.sh` dnsmasq `ipset=` directives need the stripped form (`ipset=/anthropic.com/allowed_ips` -- dnsmasq natively treats bare domains as matching all subdomains). A `stripWildcard` FuncMap function handles this in templates. The `warmup-dns.sh` script reads `dynamic-domains.conf`, so if that file contains stripped names, warmup works automatically with no additional handling.
- **Single-quoted interpolation in shell templates**: All domain name interpolation in `init-firewall.sh.tmpl` uses single quotes (e.g., `'{{.Name}}'`) to prevent shell expansion. This is the second layer of defense alongside input validation.

### Files to Create

- `internal/firewall/validate.go` -- Domain name validation function (`ValidateDomain`)
- `internal/firewall/validate_test.go` -- Tests for domain validation
- `internal/render/templates/init-firewall.sh.tmpl` -- Template for the main firewall initialization script
- `internal/render/templates/warmup-dns.sh.tmpl` -- Template for DNS warmup script (static, no parameterization)
- `internal/render/templates/dynamic-domains.conf.tmpl` -- Template for dnsmasq dynamic domain config
- `internal/render/firewall.go` -- Rendering functions: parse embedded templates, execute against GenerationConfig
- `internal/render/firewall_test.go` -- Tests for template rendering

### Files to Modify

- `internal/firewall/merge.go` -- Add domain validation call for user extras; return error on invalid input
- `internal/render/render.go` -- Update `Merge` signature to propagate validation errors from `firewall.Merge`
- `internal/render/doc.go` -- Update package doc to mention template rendering (minor)

### Detailed Steps

#### Step 1: Add domain name validation to `internal/firewall/`

**File**: `internal/firewall/validate.go`

Add a `ValidateDomain(name string) error` function that validates a domain name against a strict RFC 1123 pattern. The function:

- Accepts: `example.com`, `sub.example.com`, `a-b.example.com`, `*.example.com` (wildcard)
- Rejects: empty strings, strings with spaces, shell metacharacters (`;`, `$`, `` ` ``, `|`, `&`, `(`, `)`, `{`, `}`, `\`, `'`, `"`, `>`, `<`, `!`, `#`, `~`), strings not matching the DNS hostname pattern
- Uses a compiled `regexp.Regexp` at package level for efficiency
- Pattern: `^(\*\.)?([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`
- Labels must be 1-63 characters, total name must be 1-253 characters
- Returns a descriptive error message including the invalid input (for CLI error reporting)

**File**: `internal/firewall/validate_test.go`

Table-driven tests covering:
- Valid bare domains: `example.com`, `sub.example.com`, `a-b.example.com`
- Valid wildcard domains: `*.example.com`, `*.sub.example.com`
- Invalid: empty string, leading/trailing hyphens (`-example.com`), shell injection attempts (`; rm -rf /`, `$(cmd)`, `` `cmd` ``), spaces, consecutive dots, IP addresses (questionable -- may allow), bare `*`
- Edge cases: single-label domains (`localhost`), maximum label length (63 chars), maximum total length (253 chars)

#### Step 2: Update `firewall.Merge` to validate user extras

**File**: `internal/firewall/merge.go`

Change the `Merge` signature from:
```go
func Merge(stacks []stack.StackID, userExtras []string) MergedDomains
```
to:
```go
func Merge(stacks []stack.StackID, userExtras []string) (MergedDomains, error)
```

In the user extras loop (Step 3 of the current code), after trimming and lowercasing, call `ValidateDomain(name)`. If validation fails, return a zero `MergedDomains` and the error. Registry domains are trusted (curated by us) and do not need runtime validation.

This is a breaking change to the `Merge` signature. Update all callers:
- `internal/render/render.go` (`Merge` function) -- propagate the error
- `internal/firewall/merge_test.go` -- update all test call sites to handle the second return value
- `internal/render/render_test.go` -- update all test call sites to handle the error

Add new tests in `merge_test.go`:
- `TestMerge_InvalidUserExtra_ShellInjection` -- verify `Merge(nil, []string{"; rm -rf /"})` returns an error
- `TestMerge_InvalidUserExtra_CommandSubstitution` -- verify `Merge(nil, []string{"$(whoami)"})` returns an error
- `TestMerge_ValidUserExtras_StillWork` -- verify that valid domain names continue to pass after validation is added

#### Step 3: Create template directory and `dynamic-domains.conf.tmpl`

**File**: `internal/render/templates/dynamic-domains.conf.tmpl`

The simplest parameterized template. One domain per line from `Domains.Dynamic`. Include a header comment explaining the file's purpose. Uses the `stripWildcard` FuncMap helper to output bare domain names.

Template data source: `{{range .Domains.Dynamic}}` iterating over `firewall.Domain` structs.

Output format:
```
# Dynamic domains managed by dnsmasq for periodic re-resolution.
# Generated by ccbox -- do not edit manually.
{{range .Domains.Dynamic}}
{{stripWildcard .Name}} # {{.Rationale}}
{{end}}
```

Key detail: `{{stripWildcard .Name}}` converts `*.anthropic.com` to `anthropic.com`. For domains without a wildcard prefix, it passes through unchanged. This ensures `warmup-dns.sh` can `dig` every line without encountering unresolvable wildcard syntax.

#### Step 4: Create `warmup-dns.sh.tmpl`

**File**: `internal/render/templates/warmup-dns.sh.tmpl`

A static template (no parameterization). It reads `dynamic-domains.conf` at runtime, resolves each domain through dnsmasq using `dig`, which triggers dnsmasq's ipset integration to add resolved IPs to the firewall allowlist.

Key script behavior:
- `#!/usr/bin/env bash` with `set -euo pipefail`
- Uses `SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"` to resolve the config file relative to itself
- For each non-comment, non-empty line in `dynamic-domains.conf`: run `dig +short <domain> @127.0.0.1` to force resolution through local dnsmasq
- Log each resolution for debugging
- This script runs once at container start to "warm" the DNS cache and populate ipset

Design note: Even though this is "static" (no Go template variables), it is still a `.tmpl` file rendered through `text/template` for consistency with the other templates. This means the rendering pipeline treats all three files uniformly.

#### Step 5: Create `init-firewall.sh.tmpl`

**File**: `internal/render/templates/init-firewall.sh.tmpl`

The most complex template. This is the main firewall initialization script that runs as root inside the container (via a devcontainer lifecycle hook). It sets up iptables rules to implement a default-deny network policy with explicit allowlisting.

Script structure (in order):

1. **Shebang and strict mode**: `#!/usr/bin/env bash`, `set -euo pipefail`
2. **Preserve Docker DNS**: Detect the Docker-provided DNS server (from `/etc/resolv.conf`), add an iptables ACCEPT rule for it before the DROP policy. Without this, DNS resolution breaks entirely.
3. **Preserve loopback**: Allow all traffic on `lo` interface.
4. **Preserve established connections**: `iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT`
5. **Create ipset**: `ipset create allowed_ips hash:ip` for storing resolved IPs.
6. **Resolve static domains**: Loop over `{{range .Domains.Static}}` domains, resolve each with `dig +short '{{.Name}}'` (single-quoted), add all returned A records to the ipset. Static domains are resolved once here and cached.
7. **Install and configure dnsmasq**: Write dnsmasq config that hooks into ipset -- when dnsmasq resolves a dynamic domain, it automatically adds the result to the `allowed_ips` ipset. Use `ipset=/{{stripWildcard .Name}}/allowed_ips` directives for each dynamic domain. The `stripWildcard` helper ensures `*.anthropic.com` becomes `ipset=/anthropic.com/allowed_ips` -- dnsmasq natively treats this as matching the base domain and all subdomains.
8. **Configure iptables rules**:
   - Allow DNS to localhost (dnsmasq) on port 53
   - Allow all traffic to IPs in the `allowed_ips` ipset
   - Default DROP policy for OUTPUT chain
9. **Run warmup**: Execute `warmup-dns.sh` to pre-resolve all dynamic domains.
10. **Verification**: Test connectivity to a known-good domain (e.g., `github.com`) and log success/failure.

Template variables used:
- `{{range .Domains.Static}}` -- iterate static domains, access `.Name` and `.Rationale`
- `{{range .Domains.Dynamic}}` -- iterate dynamic domains for dnsmasq config
- `{{stripWildcard .Name}}` -- strip `*.` prefix for dnsmasq ipset directives

Shell injection defense: All domain name interpolations use single quotes in the shell template (e.g., `dig +short '{{.Name}}'`). Combined with the `ValidateDomain` input validation, this provides two independent barriers against injection.

#### Step 6: Create `internal/render/firewall.go`

**File**: `internal/render/firewall.go`

This file contains:

1. **Embed directive**: `//go:embed templates/init-firewall.sh.tmpl templates/warmup-dns.sh.tmpl templates/dynamic-domains.conf.tmpl` with `var templatesFS embed.FS`
2. **FuncMap with `stripWildcard`**: A `template.FuncMap` containing the `stripWildcard` function that strips a leading `*.` prefix from a domain name, returning the bare domain. Implementation: `strings.TrimPrefix(name, "*.")`.
3. **Template parsing** at package level: Parse all three templates from the embedded FS with the FuncMap. Use `template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.tmpl")`. Store the parsed `*template.Template` in a package-level var via `template.Must(...)`.
4. **`FirewallFiles` type**: A struct holding the three rendered outputs:
   ```go
   type FirewallFiles struct {
       InitFirewall   []byte // init-firewall.sh content
       WarmupDNS      []byte // warmup-dns.sh content
       DynamicDomains []byte // dynamic-domains.conf content
   }
   ```
5. **`RenderFirewall(cfg GenerationConfig) (FirewallFiles, error)`**: Executes each template against `cfg`, captures output into `bytes.Buffer`, returns the struct. Returns an error if any template execution fails.

Design notes:
- Parse templates once at package level via `template.Must(...)`. This fails fast at program startup if templates have syntax errors, which is the correct behavior for embedded templates that should always be valid.
- The function returns `[]byte` not `string` because downstream file-writing code (`os.WriteFile`) wants bytes.
- No file I/O in this function -- pure transformation from config to rendered bytes. File writing is the responsibility of the orchestrator (`ccbox init` command).
- The `stripWildcard` FuncMap function is deliberately minimal: just `strings.TrimPrefix`. It does not validate input (validation happens upstream in `firewall.Merge`).

#### Step 7: Create `internal/render/firewall_test.go`

**File**: `internal/render/firewall_test.go`

Testing strategy -- a mix of structural validation, content spot-checks, and security-specific tests:

**Test 1: `TestRenderFirewall_NoError`**
- Call `RenderFirewall` with a `GenerationConfig` from `Merge([]stack.StackID{stack.Go}, nil)`.
- Assert no error returned.
- Assert all three fields in `FirewallFiles` are non-empty (`len > 0`).

**Test 2: `TestRenderFirewall_InitFirewall_ContainsStaticDomains`**
- Merge Go+Node stacks.
- Render firewall files.
- Assert `InitFirewall` output contains each static domain name from `cfg.Domains.Static` (e.g., "github.com", "api.github.com", "registry.npmjs.org"). Use `bytes.Contains` for spot-checks.
- This verifies the `{{range .Domains.Static}}` loop works.

**Test 3: `TestRenderFirewall_InitFirewall_ContainsDynamicDomains`**
- Merge Go stack.
- Assert `InitFirewall` output contains dnsmasq ipset directives for dynamic domains (e.g., `ipset=/proxy.golang.org/allowed_ips`).
- This verifies the `{{range .Domains.Dynamic}}` loop generates dnsmasq config lines.

**Test 4: `TestRenderFirewall_DynamicDomains_ContainsDomainNames`**
- Merge Go+Node stacks.
- Assert `DynamicDomains` output contains each domain from `cfg.Domains.Dynamic` (with wildcards stripped).
- Assert it does NOT contain static domain names (e.g., "github.com" should not appear since it is Static).

**Test 5: `TestRenderFirewall_DynamicDomains_ContainsRationale`**
- Verify the generated `dynamic-domains.conf` includes rationale comments.

**Test 6: `TestRenderFirewall_WarmupDNS_IsStatic`**
- Render with two different configs (Go-only vs Go+Node).
- Assert `WarmupDNS` output is identical for both (since it is a static template with no variables).

**Test 7: `TestRenderFirewall_EmptyDomains`**
- Construct a `GenerationConfig` with empty Static and Dynamic slices (but non-nil).
- Assert rendering succeeds without error.
- Assert outputs are still valid (shebang present, no template rendering artifacts like `<no value>`).

**Test 8: `TestRenderFirewall_InitFirewall_ScriptStructure`**
- Assert `InitFirewall` starts with `#!/usr/bin/env bash`.
- Assert it contains `set -euo pipefail`.
- Assert it contains key structural markers: `ipset create`, `iptables`, `dnsmasq`.
- This is a structural smoke test that the script is well-formed.

**Test 9: `TestRenderFirewall_WarmupDNS_ScriptStructure`**
- Assert starts with shebang.
- Assert contains `dig` command.
- Assert contains reference to `dynamic-domains.conf`.

**Test 10: `TestRenderFirewall_AllStacks`**
- Use all five stacks + user extras.
- Structural assertion: every domain from `cfg.Domains.Dynamic` appears in the `DynamicDomains` output (with wildcards stripped).
- Structural assertion: every domain from `cfg.Domains.Static` appears somewhere in the `InitFirewall` output.

**Test 11: `TestRenderFirewall_WildcardDomainHandling`**
- Merge with no extra stacks (only AlwaysOn, which includes `*.anthropic.com`).
- Render firewall files.
- Assert `DynamicDomains` output contains `anthropic.com` (bare, no `*.` prefix).
- Assert `DynamicDomains` output does NOT contain `*.anthropic.com` (literal wildcard form).
- Assert `InitFirewall` output contains `ipset=/anthropic.com/allowed_ips` (dnsmasq directive with stripped wildcard).
- Assert `InitFirewall` output does NOT contain `ipset=/*.anthropic.com/` (the raw wildcard form should not appear in ipset directives).

**Test 12: `TestRenderFirewall_InitFirewall_SingleQuotedDomains`**
- Merge Go stack.
- Render firewall files.
- Assert `InitFirewall` output contains at least one occurrence of a single-quoted domain in a `dig` command context (e.g., the string `'github.com'` appears). This verifies the shell injection defense layer in templates.

#### Step 8: Update `internal/render/doc.go`

Minor update to mention that the package now renders templates in addition to merging configs.

### Template Content Design

#### init-firewall.sh

The generated script follows standard Linux container firewall patterns:

- Uses `iptables` (not `nftables`) for broad container compatibility (Docker, Podman, devcontainer runtimes all support iptables).
- Uses `ipset` for efficient IP matching -- O(1) lookup instead of O(n) iptables rules.
- Uses `dnsmasq` as a local DNS forwarder with `ipset` integration: when dnsmasq resolves a domain, it can automatically add the resolved IP to a named ipset. This handles CDN domains whose IPs rotate.
- The `dig +short` calls resolve to A records. The script handles multiple A records per domain (CDNs often return several).
- Wildcard domains (e.g., `*.anthropic.com`) are handled by stripping the `*.` prefix via the `stripWildcard` FuncMap helper. In dnsmasq config, `ipset=/anthropic.com/allowed_ips` natively matches the base domain and all subdomains. In `dig` commands, the bare `anthropic.com` form is used since dig cannot resolve wildcard DNS names.
- All domain name interpolations are single-quoted in shell context (e.g., `dig +short '{{.Name}}'`) to prevent shell expansion. This is a defense-in-depth measure alongside upstream `ValidateDomain` input validation.

#### dynamic-domains.conf

Plain text, one domain per line with inline rationale comments. Wildcard domains have their `*.` prefix stripped (via `stripWildcard` in the template) so that `dig` in `warmup-dns.sh` can resolve them. This file is read by `warmup-dns.sh` and also serves as human-readable documentation of which dynamic domains are allowed.

#### warmup-dns.sh

Reads `dynamic-domains.conf`, strips comments and blank lines, resolves each domain through the local dnsmasq (port 53 on 127.0.0.1). The resolution triggers dnsmasq's ipset integration, populating the `allowed_ips` set. This ensures dynamic domain IPs are available in the ipset before the user needs network access. Because `dynamic-domains.conf` already has wildcards stripped, no additional wildcard handling is needed here.

### Testing Strategy

- **TDD**: Write tests first, verify they fail, then implement templates.
- **No golden files**: Avoid brittle snapshot tests. Instead, use structural assertions (contains specific strings, starts with shebang, has N lines matching a pattern) and content spot-checks.
- **Registry-computed expectations**: Where possible, compute expected values from the firewall registry rather than hardcoding. For example, verify that every domain in `cfg.Domains.Static` appears in the rendered init-firewall.sh, rather than hardcoding a specific list.
- **Template syntax validation**: The `template.Must` at package level ensures syntax errors are caught at startup. Tests verify the templates execute without runtime errors.
- **Edge case: empty domains**: Test that templates produce valid output even when no stacks are detected (only always-on domains) or when domain lists are empty.
- **Wildcard domain rendering**: Dedicated test (`TestRenderFirewall_WildcardDomainHandling`) verifies that `*.anthropic.com` is correctly stripped to `anthropic.com` in both `dynamic-domains.conf` and `init-firewall.sh` dnsmasq directives.
- **Shell injection defense verification**: `TestRenderFirewall_InitFirewall_SingleQuotedDomains` verifies that domain interpolation in shell contexts uses single quotes. `TestValidateDomain_*` tests in the firewall package verify that shell metacharacters are rejected at the input validation layer.

### Open Questions

None -- all design decisions are grounded in the existing codebase patterns and standard Linux container firewall tooling. The template approach (`embed.FS` + `text/template` + pure rendering functions) aligns with ADR-0002 and the sibling beans' expected patterns. The shell injection mitigation uses two independent defense layers (input validation + single-quoting) which is standard practice for generated shell scripts.

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
| refine | done | 2 | 2026-04-02 |
| challenge | approved | 2 | 2026-04-02 |
| implement | in-progress | 1 | 2026-04-02 |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |