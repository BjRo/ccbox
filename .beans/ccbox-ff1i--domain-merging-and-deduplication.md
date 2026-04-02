---
# ccbox-ff1i
title: Domain merging and deduplication
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:35:46Z
updated_at: 2026-04-02T12:53:39Z
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

- [x] Tests written (TDD)
- [x] No TODO/FIXME/HACK/XXX in new code
- [x] Lint passes (`golangci-lint run ./...`)
- [x] Tests pass (`go test ./...`)
- [x] Branch pushed
- [x] PR created
- [x] Automated code review passed
- [x] Review feedback worked in
- [ ] ADR written (if architectural changes)

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-02 |
| challenge | complete | 1 | 2026-04-02 |
| implement | complete | 1 | 2026-04-02 |
| pr | complete | 1 | 2026-04-02 |
| review | complete | 1 | 2026-04-02 |
| codify | in-progress | 1 | 2026-04-02 |

## Implementation Plan

### Approach

Add a `Merge` function to `internal/firewall/` that accepts a list of `stack.StackID` values (the detected/selected stacks) and a list of user-provided extra domain strings. It collects always-on domains, per-stack domains from the firewall registry, and user extras, deduplicates by domain name (first occurrence wins for category/rationale), and returns two sorted, deduplicated slices: one for Static domains and one for Dynamic domains.

The bridging between `stack.StackID` and `firewall.Stack` is straightforward: both are string types with identical underlying values for the 5 real stacks (`"go"`, `"node"`, `"python"`, `"rust"`, `"ruby"`). The `Merge` function accepts `stack.StackID` values and converts them to `firewall.Stack` via a simple type conversion `firewall.Stack(id)` internally.

User-provided extra domains default to `Dynamic` category with a standard rationale string, since the user cannot know whether a domain uses static or rotating IPs, and Dynamic is the safer default (handles CDN/load-balancer IP changes via dnsmasq re-resolution).

The result type is a `MergedDomains` struct containing two `[]Domain` slices (Static and Dynamic), both sorted by domain name. This gives callers structured access to the full domain metadata (name, category, rationale) for template rendering.

The function lives in `internal/firewall/` rather than a new package because it operates entirely on firewall types and the firewall registry. The `firewall` package gains a new import of `internal/stack` (for the `stack.StackID` type), which is fine because the dependency flows from behavior package to data package, consistent with ADR-0004.

### Files to Create/Modify

- `internal/firewall/merge.go` -- New file containing the `MergedDomains` struct and `Merge` function
- `internal/firewall/merge_test.go` -- New file containing comprehensive TDD tests for the merge logic

### Types

```go
// MergedDomains holds the final deduplicated domain lists, split by category,
// ready for template rendering.
type MergedDomains struct {
    Static  []Domain // Domains with stable IPs, resolved once at firewall init
    Dynamic []Domain // Domains with rotating IPs, managed by dnsmasq
}
```

### Function Signature

```go
// Merge combines always-on domains, per-stack domains for the given stacks,
// and user-provided extra domains into two deduplicated, sorted lists (Static
// and Dynamic). Unknown stack IDs are silently skipped. User extras are
// classified as Dynamic by default.
func Merge(stacks []stack.StackID, userExtras []string) MergedDomains
```

### Steps

1. **Define `MergedDomains` type in `internal/firewall/merge.go`**
   - Struct with `Static []Domain` and `Dynamic []Domain` fields.
   - Doc comment explaining the split and intended consumers (template rendering for init-firewall.sh and dynamic-domains.conf).

2. **Implement the `Merge` function in `internal/firewall/merge.go`**
   - Initialize a `seen` map (`map[string]bool`) for deduplication by domain name.
   - Initialize two `[]Domain` accumulators: `staticDomains` and `dynamicDomains`.
   - Define a local helper `addDomain(d Domain)` that checks `seen`, inserts into the correct accumulator based on `d.Category`, and marks the name as seen. First occurrence wins -- if a domain appears in always-on AND a per-stack list, the always-on entry (processed first) takes precedence.
   - Step 1: Collect always-on domains. Call `ForStack(AlwaysOn)` and iterate, calling `addDomain` for each.
   - Step 2: Collect per-stack domains. For each `stack.StackID` in the input slice, convert to `firewall.Stack(id)` and call `ForStack`. If not found (unknown stack), skip silently. Iterate the returned domains, calling `addDomain` for each.
   - Step 3: Collect user extras. For each string in `userExtras`, skip empty strings (after trimming whitespace), create a `Domain{Name: name, Category: Dynamic, Rationale: "User-specified domain"}`, and call `addDomain`.
   - Step 4: Sort both accumulator slices by `Domain.Name` using `slices.SortFunc` with `strings.Compare`.
   - Return `MergedDomains{Static: staticDomains, Dynamic: dynamicDomains}`. Return empty (non-nil) slices if no domains in a category.

3. **Edge cases handled in the implementation**
   - Empty `stacks` slice: only always-on domains + user extras appear.
   - Empty `userExtras` slice: no user domains added.
   - Both empty: only always-on domains appear (the function always includes AlwaysOn).
   - Duplicate stack IDs in input: no effect because domains are deduplicated by name.
   - Unknown stack IDs: silently skipped (ForStack returns false).
   - User extra that duplicates a registry domain: first occurrence (from registry) wins, user extra is skipped.
   - Empty string or whitespace-only user extras: skipped.
   - User extras with leading/trailing whitespace: trimmed before processing.

### Testing Strategy

All tests in `internal/firewall/merge_test.go`, package `firewall` (internal access).

**Test 1: `TestMerge_AlwaysOnIncluded`**
- Call `Merge(nil, nil)` with no stacks and no user extras.
- Verify that all 5 always-on domains appear in the result, split correctly by category (3 Static, 2 Dynamic based on current registry: github.com and api.github.com and sentry.io and statsig.com are Static; *.anthropic.com is Dynamic).
- Actually: api.github.com (Static), github.com (Static), sentry.io (Static), statsig.com (Static) = 4 static; *.anthropic.com (Dynamic) = 1 dynamic.
- Verify sorted order within each list.

**Test 2: `TestMerge_SingleStack`**
- Call `Merge([]stack.StackID{stack.Go}, nil)`.
- Verify always-on domains are present.
- Verify Go-specific domains (proxy.golang.org, sum.golang.org, storage.googleapis.com -- all Dynamic) are present.
- Verify no domains from other stacks (e.g., no registry.npmjs.org).

**Test 3: `TestMerge_MultipleStacks`**
- Call `Merge([]stack.StackID{stack.Go, stack.Node}, nil)`.
- Verify domains from both Go and Node are present alongside always-on.
- Verify deduplication across stacks (no duplicate domain names in output).
- Verify correct Static/Dynamic split.

**Test 4: `TestMerge_UserExtras`**
- Call `Merge(nil, []string{"custom.example.com", "another.example.com"})`.
- Verify user extras appear in the Dynamic list with `Category: Dynamic` and `Rationale: "User-specified domain"`.
- Verify they are sorted.

**Test 5: `TestMerge_UserExtraDuplicatesRegistry`**
- Call `Merge([]stack.StackID{stack.Go}, []string{"proxy.golang.org"})`.
- Verify `proxy.golang.org` appears exactly once (from registry, not from user extras).
- Verify the retained entry has the registry's rationale, not "User-specified domain".

**Test 6: `TestMerge_UserExtraDuplicatesAlwaysOn`**
- Call `Merge(nil, []string{"github.com"})`.
- Verify `github.com` appears exactly once in the Static list (from AlwaysOn), not in Dynamic.
- The registry entry wins because AlwaysOn is processed first.

**Test 7: `TestMerge_DeduplicateUserExtras`**
- Call `Merge(nil, []string{"dup.example.com", "dup.example.com"})`.
- Verify `dup.example.com` appears exactly once.

**Test 8: `TestMerge_UnknownStackSkipped`**
- Call `Merge([]stack.StackID{"elixir"}, nil)`.
- Verify only always-on domains appear (no error, unknown stack silently skipped).

**Test 9: `TestMerge_EmptyUserExtrasSkipped`**
- Call `Merge(nil, []string{"", "  ", "valid.example.com"})`.
- Verify only `valid.example.com` appears from user extras (empty and whitespace-only strings skipped).

**Test 10: `TestMerge_SortedOutput`**
- Call `Merge([]stack.StackID{stack.Go, stack.Node, stack.Python}, nil)`.
- Verify both Static and Dynamic slices are sorted by Domain.Name.

**Test 11: `TestMerge_DuplicateStackIDs`**
- Call `Merge([]stack.StackID{stack.Go, stack.Go}, nil)`.
- Verify no duplicate domains in result (same count as single Go).

**Test 12: `TestMerge_AllStacks`**
- Call `Merge` with all 5 stacks and no user extras.
- Verify total unique domain count equals the unique domain count across all registry entries.
- Verify no duplicates by checking `len(Static) + len(Dynamic)` against a manually counted set.

**Test 13: `TestMerge_UserExtraWhitespaceTrimmed`**
- Call `Merge(nil, []string{"  trimmed.example.com  "})`.
- Verify the domain name in the result is `"trimmed.example.com"` (no leading/trailing spaces).

### Challenge Feedback (incorporated)

1. **MergedDomains ownership semantics**: Add doc comment on `MergedDomains` stating callers own the returned slices and may mutate them freely. No code change needed since slices are freshly allocated.
2. **Test brittleness**: Tests should verify structural invariants rather than hardcoded counts. Use `ForStack()` programmatically to compute expected sets, then compare against `Merge` output. This tests merge logic without encoding registry contents.

### Open Questions

None. The approach is straightforward and all design decisions are grounded in existing patterns and the bean description.