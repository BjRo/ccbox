---
# ccbox-tv1t
title: Multi-stack merging logic
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:34:25Z
updated_at: 2026-04-02T15:08:49Z
parent: ccbox-2n15
---

## Description
When multiple stacks are detected, merge their metadata:
- Combine all runtimes into a single mise.toml
- Merge LSP server installations in Dockerfile
- Merge LSP plugin lists in claude-user-settings.json
- Union all default + dynamic domain allowlists
- Deduplicate domains

Input: list of detected Stack objects + user-selected extras
Output: a merged `GenerationConfig` struct used by the template engine

## Implementation Plan

### Approach

Create the `GenerationConfig` struct and a `Merge` function in the `internal/render` package. This is the natural home because `GenerationConfig` is the render package's input data model -- the type that feeds into Go templates to produce Dockerfiles, devcontainer.json, mise.toml, and Claude Code settings. The merge function collects stack metadata from `internal/stack` and delegates domain merging to the existing `firewall.Merge()`.

The function signature: `Merge(stacks []stack.StackID, userExtraDomains []string) (GenerationConfig, error)`. It accepts detected stack IDs (from `detect.Detect`) and user-provided extra domains (from CLI flags or wizard), returning the fully merged config. An error is returned if any stack ID is not found in the registry, to prevent silent misconfiguration (unlike `firewall.Merge` which silently skips unknown stacks -- that behavior is appropriate for domains where an unknown stack simply means "no extra domains", but for the full config an unrecognized stack ID likely indicates a bug).

### Files to Create/Modify

- `internal/render/render.go` -- Replace the current doc-only placeholder with the `GenerationConfig` type, supporting types (`MergedRuntime`, `MergedLSP`), and the `Merge` function
- `internal/render/render_test.go` -- Comprehensive tests for the Merge function
- `internal/render/doc.go` -- Extract the package doc comment to its own file (consistent with `stack` and `detect` packages which use `doc.go`)

### Steps

1. **Create `internal/render/doc.go`** -- Move the package doc comment to a dedicated file, following the pattern in `internal/stack/doc.go` and `internal/detect/doc.go`. Update the doc to describe the dual responsibility: merge logic and template rendering.

2. **Define the `GenerationConfig` struct in `internal/render/render.go`**

   The struct captures everything the template engine will need:

   ```go
   // MergedRuntime represents a single tool entry for mise.toml.
   type MergedRuntime struct {
       Tool    string // mise tool name, e.g. "go", "node"
       Version string // Version strategy, e.g. "latest", "lts"
   }

   // MergedLSP represents a single language server to install and configure.
   type MergedLSP struct {
       Package    string // LSP server package name
       InstallCmd string // Full install command for Dockerfile
       Plugin     string // Claude Code plugin identifier
   }

   // GenerationConfig holds the fully merged, deduplicated configuration
   // produced by combining multiple detected stacks. It is the single input
   // to the template rendering pipeline.
   type GenerationConfig struct {
       Stacks   []stack.StackID     // Detected stack IDs, sorted
       Runtimes []MergedRuntime     // Merged runtime entries for mise.toml, sorted by Tool
       LSPs     []MergedLSP         // Merged LSP servers for Dockerfile + settings, sorted by Package
       Domains  firewall.MergedDomains // Merged domain allowlists (static + dynamic)
   }
   ```

   Key design decisions:
   - `MergedRuntime` and `MergedLSP` are separate types rather than reusing `stack.Runtime` and `stack.LSP` to keep the render package's API independent of the stack package's internal types. If the stack types evolve (e.g., adding fields irrelevant to rendering), `GenerationConfig` remains stable. However, given the project's early stage and YAGNI, these types will be structurally identical to `stack.Runtime` and `stack.LSP` for now. If this feels like premature indirection during challenge, the alternative is to directly embed `stack.Runtime` and `stack.LSP` slices in `GenerationConfig`.
   - `Domains` uses the existing `firewall.MergedDomains` type directly since domain merging is fully delegated to `firewall.Merge()`.
   - All slice fields are sorted for deterministic template output.

3. **Implement the `Merge` function in `internal/render/render.go`**

   The function follows these steps:
   1. Validate all stack IDs exist in the registry. Return an error with the first unknown ID: `fmt.Errorf("render: unknown stack %q", id)`.
   2. Look up each stack via `stack.Get(id)` and collect runtimes and LSPs.
   3. Deduplicate runtimes by `Tool` name (first-occurrence-wins, matching the domain merging precedent). This handles the unlikely case of two stacks declaring the same mise tool.
   4. Deduplicate LSPs by `Package` name (first-occurrence-wins).
   5. Sort runtimes by `Tool`, LSPs by `Package`.
   6. Delegate domain merging to `firewall.Merge(stacks, userExtraDomains)`.
   7. Return the assembled `GenerationConfig`.

   The deduplication logic is straightforward since the current registry has no overlapping runtimes or LSPs, but the code should handle it correctly for future-proofing and to match the principled approach used in `firewall.Merge`.

4. **Write tests in `internal/render/render_test.go`**

   Following the project's testing patterns (structural invariants from registry + hardcoded spot-checks):

   - **TestMerge_SingleStack** -- Merge with just `stack.Go`. Assert one runtime (tool="go"), one LSP (package="gopls"), and domains match `firewall.Merge([]stack.StackID{stack.Go}, nil)`.
   - **TestMerge_MultipleStacks** -- Merge `stack.Go` and `stack.Node`. Assert two runtimes, two LSPs. Spot-check that both "go" and "node" tools are present, both "gopls" and "typescript" plugins are present.
   - **TestMerge_AllStacks** -- Merge all 5 stacks. Structural: runtime count == number of unique tools across all stacks, LSP count == number of unique packages. Verify no duplicates.
   - **TestMerge_EmptyStacks** -- Merge with empty stack list. Assert empty runtimes, empty LSPs, but domains still include always-on entries (from `firewall.Merge`). Verify non-nil empty slices for Runtimes and LSPs.
   - **TestMerge_UnknownStack** -- Merge with `StackID("elixir")`. Assert error is returned and contains "unknown stack".
   - **TestMerge_DuplicateStackIDs** -- Merge `[Go, Go]`. Assert same result as single Go (no duplicated runtimes/LSPs).
   - **TestMerge_SortedOutput** -- Merge multiple stacks. Assert Runtimes sorted by Tool, LSPs sorted by Package, Stacks sorted alphabetically.
   - **TestMerge_UserExtraDomains** -- Merge with user extras passed through. Assert they appear in `Domains.Dynamic`.
   - **TestMerge_StacksFieldMatchesInput** -- Assert `GenerationConfig.Stacks` contains exactly the deduplicated, sorted input stack IDs.
   - **TestMerge_RuntimesMatchRegistry** -- For each input stack, assert its runtime appears in the result. Structural: count unique tools in registry for input stacks, compare to `len(result.Runtimes)`.
   - **TestMerge_LSPsMatchRegistry** -- Same pattern for LSPs.

### Testing Strategy

- All tests use table-driven style where appropriate (e.g., single-stack tests per stack).
- Structural invariants computed from the stack registry (via `stack.Get()`) to avoid hardcoded counts that break when registry data changes.
- Hardcoded spot-checks for well-known values (e.g., "gopls" in Go, "node" runtime for Node).
- Domain assertions delegate verification to comparing against `firewall.Merge()` output, since that function is already thoroughly tested.
- No filesystem I/O needed -- this is pure in-memory data transformation.
- Verify non-nil empty slices (not nil) for zero-element cases, since Go templates behave differently with nil vs empty slices.

### Dependency Graph

```
internal/render  -->  internal/stack     (for Stack metadata)
internal/render  -->  internal/firewall  (for MergedDomains type + Merge function)
```

This follows ADR-0004: behavior packages import the data registry, never the reverse. The `render` -> `firewall` dependency is acceptable since `render` is a leaf consumer that orchestrates the final config.

### Open Questions

None -- the existing patterns in `firewall.Merge` provide clear precedent for the approach. The only design judgment (own types vs reusing `stack.Runtime`/`stack.LSP`) is noted above and either choice is defensible at this stage.

## Challenge Report

**Scope: SMALL CHANGE** (3 files)

### Scope Assessment

| Metric | Value | Threshold |
|--------|-------|-----------|
| Files | 3 | >15 = recommend split |

### Findings

#### Go Engineer

> **Finding 1: MergedRuntime and MergedLSP are premature abstractions** (severity: WARNING)
>
> Step 2 introduces `MergedRuntime` and `MergedLSP` as render-package-local types that are field-for-field identical to `stack.Runtime` and `stack.LSP`. The plan acknowledges this tension ("If this feels like premature indirection during challenge, the alternative is to directly embed..."). It does feel like premature indirection. The justification -- that `stack.Runtime` might evolve with fields irrelevant to rendering -- is speculative. Today `stack.Runtime` has exactly two fields (`Tool`, `Version`) and both are consumed by templates. Creating parallel types means every future field addition to the stack types requires a conscious decision about the render types, and the conversion code from `stack.Runtime` to `MergedRuntime` is pure boilerplate that adds lines without adding safety.
>
> This is exactly what the engineering calibration flags as over-engineering: "premature abstractions, unnecessary indirection. The simpler approach wins." Meanwhile, the plan already uses `firewall.MergedDomains` directly rather than wrapping it, which demonstrates the right instinct -- apply the same principle to runtimes and LSPs.
>
> **Option A (recommended):** Use `[]stack.Runtime` and `[]stack.LSP` directly in `GenerationConfig`. This eliminates two type definitions, removes all field-copying boilerplate, and keeps the code honest about the current coupling. If the stack types later grow fields irrelevant to rendering, that is the right time to introduce a render-local projection type -- with a concrete reason driving the shape.
>
> **Option B:** Keep `MergedRuntime` and `MergedLSP` but define them as type aliases (`type MergedRuntime = stack.Runtime`) to get the distinct naming for documentation purposes without the conversion overhead. This is a middle ground but adds naming indirection for little benefit at this stage.

> **Finding 2: Step 3 does not explicitly deduplicate the Stacks field** (severity: WARNING)
>
> Step 3's algorithm (substeps 1-7) validates, collects runtimes/LSPs, deduplicates runtimes by Tool, deduplicates LSPs by Package, sorts, merges domains, and returns. But it never mentions deduplicating the `Stacks` field itself. The input `stacks []stack.StackID` could contain duplicates (e.g., `[Go, Go]`), and `TestMerge_DuplicateStackIDs` (step 4) explicitly tests for this. The runtimes and LSPs are deduplicated via their Tool/Package keys, but `GenerationConfig.Stacks` would contain `[go, go]` unless the implementation adds an explicit dedup step. `TestMerge_StacksFieldMatchesInput` says "deduplicated, sorted input stack IDs", confirming the intent -- but the algorithm description in step 3 has a gap.
>
> **Option A (recommended):** Add an explicit substep between current substeps 1 and 2: "Deduplicate input stack IDs (using a seen-set) and sort them for the `Stacks` field." This makes the algorithm description match the test expectations.
>
> **Option B:** Deduplicate during the validation loop in substep 1 -- as you iterate over stacks to validate via `stack.Get(id)`, track seen IDs and skip duplicates. This is slightly more efficient (single pass) and matches the first-occurrence-wins pattern used elsewhere.

### Verdict

**APPROVED** -- The plan is well-structured, follows established codebase patterns closely, and the testing strategy is thorough. Both findings are WARNINGs, not blockers. Finding 1 is the more consequential one: resolving it in favor of Option A will produce a simpler implementation with fewer lines and no loss of correctness. Finding 2 is a minor gap in the algorithm description that the tests already catch -- just needs the prose to match the intent.
