---
# ccbox-ztaa
title: Curate per-stack domain allowlists
status: in-progress
type: task
priority: high
created_at: 2026-04-02T10:35:44Z
updated_at: 2026-04-02T12:38:06Z
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

## Implementation Plan

### Approach

Implement the domain allowlist registry as a pure data package within `internal/firewall/`. The tests already exist (`firewall_test.go`) and define the exact API surface via TDD. The implementation adds three exported types (`Stack`, `Category`, `Domain`, `Allowlist`), six stack constants, and three accessor functions (`Registry`, `ForStack`, `Stacks`). All domain data is declared as a package-level registry map, with accessor functions returning defensive copies to prevent callers from mutating shared state.

This bean is scoped to the data registry only. The sibling bean `ccbox-ff1i` (Domain merging and deduplication) will consume this registry to produce merged domain lists.

### Files to Create/Modify

- `internal/firewall/firewall.go` -- Expand from stub to full implementation. Define types, constants, the package-level registry, and all three accessor functions.

No other files are created or modified. The test file already exists and should not be changed.

### Steps

1. **Define the `Stack` type and constants** -- Add `type Stack string` and six constants: `AlwaysOn`, `Go`, `Node`, `Python`, `Rust`, `Ruby`. The `AlwaysOn` pseudo-stack represents domains required by every ccbox container (Claude Code connectivity, GitHub, telemetry). Constant values must be lowercase strings that sort alphabetically to satisfy `TestStacks_Order` (which expects: `AlwaysOn` < `Go` < `Node` < `Python` < `Ruby` < `Rust`). Use the values `"always-on"`, `"go"`, `"node"`, `"python"`, `"ruby"`, `"rust"`.

2. **Define the `Category` type and constants** -- Add `type Category string` with two constants: `Static` and `Dynamic`. Static domains have stable IPs that can be resolved once and cached in iptables ipsets. Dynamic domains use CDN/rotating IPs and need dnsmasq-based periodic re-resolution.

3. **Define the `Domain` struct** -- Three fields:
   - `Name string` -- The domain name (e.g., `"proxy.golang.org"` or `"*.anthropic.com"` for wildcard).
   - `Category Category` -- Static or Dynamic.
   - `Rationale string` -- Human-readable explanation of why this domain is needed. Every domain must have a non-empty rationale per `TestRegistry_AllDomainsHaveRationale`.

4. **Define the `Allowlist` struct** -- Two fields:
   - `Stack Stack` -- Which stack this allowlist belongs to.
   - `Domains []Domain` -- The list of domains for that stack.

5. **Declare the package-level registry** -- `var registry map[Stack]Allowlist` initialized in an `init()` function or as a var literal. Populate with all six stacks and their exact domains:

   - **AlwaysOn** (5 domains):
     - `api.github.com` / Static -- GitHub API for git operations, PR creation, and Claude Code tools.
     - `github.com` / Static -- Git clone/fetch/push over HTTPS.
     - `*.anthropic.com` / Dynamic -- Claude Code API; wildcard because Anthropic uses multiple subdomains and CDN endpoints.
     - `sentry.io` / Static -- Claude Code error reporting and telemetry.
     - `statsig.com` / Static -- Claude Code feature flags and experiment configuration.

   - **Go** (3 domains, all Dynamic):
     - `proxy.golang.org` / Dynamic -- Go module proxy; uses Google CDN with rotating IPs.
     - `sum.golang.org` / Dynamic -- Go checksum database; same CDN infrastructure.
     - `storage.googleapis.com` / Dynamic -- Backing store for Go module proxy; Google Cloud CDN.

   - **Node** (3 domains):
     - `registry.npmjs.org` / Static -- npm package registry; relatively stable IPs.
     - `cdn.jsdelivr.net` / Dynamic -- CDN for npm packages; Cloudflare-backed, rotating IPs.
     - `unpkg.com` / Dynamic -- CDN for npm packages; Cloudflare-backed, rotating IPs.

   - **Python** (2 domains, both Static):
     - `pypi.org` / Static -- Python Package Index; Fastly-backed with stable anycast IPs.
     - `files.pythonhosted.org` / Static -- Package file downloads; same Fastly infrastructure.

   - **Rust** (2 domains, both Static):
     - `crates.io` / Static -- Rust package registry; stable AWS infrastructure.
     - `static.crates.io` / Static -- Crate file downloads; stable S3/CloudFront endpoint.

   - **Ruby** (2 domains, both Static):
     - `rubygems.org` / Static -- Ruby package registry; Fastly-backed with stable IPs.
     - `index.rubygems.org` / Static -- Compact index API for faster dependency resolution.

6. **Implement `Registry() map[Stack]Allowlist`** -- Returns a defensive copy of the registry map. Must copy the map itself (not deep-copy slices, since `Domain` is a value type with no pointer fields, and `Allowlist.Domains` is a slice that callers cannot mutate in a way that affects the original -- the map copy gives each caller a distinct `Allowlist` value whose `Domains` slice header is independent). The test `TestRegistry_ReturnsDefensiveCopy` validates this by deleting a key from one copy and checking a second copy still has it. Implementation: iterate the registry, copy each key-value pair into a new map, return it.

7. **Implement `ForStack(s Stack) (Allowlist, bool)`** -- Looks up a single stack in the registry. Returns the `Allowlist` and `true` if found, zero value and `false` otherwise. This is a convenience accessor for callers that only need one stack (e.g., the merging logic in `ccbox-ff1i`).

8. **Implement `Stacks() []Stack`** -- Returns a sorted slice of all stack keys in the registry. Sort lexicographically by the string value of the `Stack` type. The test `TestStacks_Order` expects: `[AlwaysOn, Go, Node, Python, Ruby, Rust]`.

### Domain Categorization Rationale

The Static vs Dynamic distinction determines how the firewall script handles each domain:

- **Static**: IP addresses are resolved once during container startup and added to an iptables ipset. Suitable for services with stable, well-known IP ranges (GitHub publishes IPs via their meta API; PyPI/RubyGems use Fastly anycast).
- **Dynamic**: Domains are configured in dnsmasq to intercept DNS queries, resolve them in real-time, and dynamically add resulting IPs to the ipset. Required for CDN-backed services (Google Cloud, Cloudflare, AWS CloudFront) where IPs rotate frequently.

Key categorization decisions:
- `*.anthropic.com` is Dynamic despite being "always-on" because Anthropic uses CDN infrastructure with rotating IPs, and the wildcard pattern requires DNS-level interception.
- All Go domains are Dynamic because they resolve through Google Cloud CDN.
- `registry.npmjs.org` is Static because npm's registry API uses stable Cloudflare enterprise IPs, unlike the CDN-served content on jsdelivr/unpkg.

### Testing Strategy

All tests are already written in `internal/firewall/firewall_test.go`. The implementation must satisfy:

- `TestRegistry_ContainsAllStacks` -- Registry has exactly 6 stacks.
- `TestRegistry_AlwaysOnDomains` -- AlwaysOn has exactly 5 domains with correct categories.
- `TestRegistry_PerStackDomains` -- Each of 5 dev stacks has exact domain names.
- `TestRegistry_AllDomainsHaveRationale` -- No domain has an empty Rationale field.
- `TestRegistry_AllDomainsHaveCategory` -- Every domain is Static or Dynamic.
- `TestRegistry_NoDuplicateDomainsWithinStack` -- No duplicate domain names within a stack.
- `TestForStack_Found` -- `ForStack(Go)` returns correct allowlist.
- `TestForStack_NotFound` -- `ForStack("elixir")` returns false.
- `TestStacks_Order` -- `Stacks()` returns alphabetically sorted slice.
- `TestRegistry_ReturnsDefensiveCopy` -- Mutating one `Registry()` result does not affect another.

Run: `go test ./internal/firewall/...`

No additional tests need to be written. The existing test suite is comprehensive and covers types, data integrity, accessor behavior, and defensive copying.

### File Organization

```
internal/firewall/
  firewall.go         -- Types, constants, registry data, accessor functions (all in one file)
  firewall_test.go    -- Tests (already exists, do not modify)
```

Everything lives in a single `firewall.go` file. At ~120 lines of implementation, splitting into multiple files would be premature. The registry data is a handful of struct literals, not a large dataset that warrants its own file.

### Open Questions

None. The test file fully specifies the API contract, the domain list is defined in the bean description, and the static/dynamic categorization follows directly from each service's infrastructure characteristics.

## Checklist

- [x] Tests written (TDD)
- [x] No TODO/FIXME/HACK/XXX comments
- [x] Lint passes
- [x] Tests pass
- [x] Branch pushed
- [x] PR created
- [x] Automated code review passed
- [x] Review feedback worked in

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-02 |
| challenge | complete | 1 | 2026-04-02 |
| implement | complete | 1 | 2026-04-02 |
| pr | complete | 1 | 2026-04-02 |
| review | complete | 1 | 2026-04-02 |
| codify | pending | | |