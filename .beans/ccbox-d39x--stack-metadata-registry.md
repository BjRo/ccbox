---
# ccbox-d39x
title: Stack metadata registry
status: in-progress
type: task
priority: high
created_at: 2026-04-02T10:34:21Z
updated_at: 2026-04-02T12:38:09Z
parent: ccbox-2n15
---

## Description
Define a registry of stack metadata. Each stack entry includes:

- **Name**: Display name (e.g., "Go", "Node/TypeScript")
- **Runtime**: mise tool name + version strategy (e.g., `go latest`, `node lts`)
- **LSP**: Language server package + install method (e.g., `gopls` via `go install`, `typescript-language-server` via npm)
- **LSP Plugin**: Claude Code plugin identifier (e.g., `gopls-lsp`, `typescript-lsp`)
- **Default domains**: Package registry domains to allowlist (e.g., `proxy.golang.org`, `registry.npmjs.org`)
- **Dynamic domains**: Domains that need dnsmasq (changing IPs)
- **VS Code extensions**: None for v1 (Claude Code only, added separately)

This registry is the single source of truth for all stack-specific behavior.

## Checklist

- [x] Tests written (TDD)
- [x] No TODO/FIXME/HACK/XXX comments
- [x] Lint passes (`golangci-lint run ./...`)
- [x] Tests pass (`go test ./...`)
- [x] Branch pushed
- [x] PR created
- [x] Automated code review passed
- [x] Review feedback worked in
- [x] All checklist items completed
- [x] User notified

## Implementation Plan

### Approach

Implement the stack metadata registry as a single-file module in `internal/stack/stack.go`. The registry is a package-level immutable map of `StackID -> Stack`, populated in an `init()` function (acceptable here since this is pure data initialization, not command wiring). The public API consists of three accessor functions (`Get`, `All`, `IDs`) that return defensive copies to prevent callers from mutating internal state. All types and data live in one file since the total code will be under 200 lines.

The existing test file (`internal/stack/stack_test.go`) was written TDD-first and already defines the full API contract. The implementation must satisfy all 9 test functions without modifying the tests.

### Files to Create/Modify

- `internal/stack/stack.go` -- Define types (`StackID`, `Runtime`, `LSP`, `Stack`), the `StackID` constants, the registry map, and accessor functions (`Get`, `All`, `IDs`). This is the only file that needs changes.

### Data Structures

```go
// StackID is a string type for type-safe stack identifiers.
type StackID string

const (
    Go     StackID = "go"
    Node   StackID = "node"
    Python StackID = "python"
    Ruby   StackID = "ruby"
    Rust   StackID = "rust"
)

// Runtime describes a mise-managed runtime.
type Runtime struct {
    Tool    string // mise tool name, e.g. "go", "node", "python"
    Version string // version strategy, e.g. "latest", "lts"
}

// LSP describes a language server and its Claude Code plugin.
type LSP struct {
    Package    string // package name, e.g. "gopls"
    InstallCmd string // installation command, e.g. "go install golang.org/x/tools/gopls@latest"
    Plugin     string // Claude Code plugin identifier, e.g. "gopls-lsp"
}

// Stack holds all metadata for a single tech stack.
type Stack struct {
    ID             StackID
    Name           string
    Runtime        Runtime
    LSP            LSP
    DefaultDomains []string
    DynamicDomains []string
    MarkerFiles    []string
}
```

**Note on StackID ordering**: The constants must be alphabetically ordered by their string values (`"go"`, `"node"`, `"python"`, `"ruby"`, `"rust"`) because `TestAll_Sorted` and `TestIDs_Sorted` assert alphabetical ordering.

### Registry Data

The registry map will contain these five entries:

**Go**
- Runtime: `{Tool: "go", Version: "latest"}`
- LSP: `{Package: "gopls", InstallCmd: "go install golang.org/x/tools/gopls@latest", Plugin: "gopls-lsp"}`
- DefaultDomains: `["proxy.golang.org", "storage.googleapis.com", "sum.golang.org"]`
- DynamicDomains: `[]string{}` (none -- Go module proxy IPs are stable)
- MarkerFiles: `["go.mod"]`

**Node/TypeScript**
- Runtime: `{Tool: "node", Version: "lts"}`
- LSP: `{Package: "typescript-language-server", InstallCmd: "npm install -g typescript-language-server typescript", Plugin: "typescript-lsp"}`
- DefaultDomains: `["registry.npmjs.org"]`
- DynamicDomains: `[]string{}` (npm CDN domains could go here in future)
- MarkerFiles: `["package.json", "tsconfig.json"]`

**Python**
- Runtime: `{Tool: "python", Version: "latest"}`
- LSP: `{Package: "pylsp", InstallCmd: "pip install python-lsp-server", Plugin: "pylsp-lsp"}`
- DefaultDomains: `["pypi.org", "files.pythonhosted.org"]`
- DynamicDomains: `[]string{}`
- MarkerFiles: `["requirements.txt", "pyproject.toml", "setup.py", "Pipfile"]`

**Ruby**
- Runtime: `{Tool: "ruby", Version: "latest"}`
- LSP: `{Package: "solargraph", InstallCmd: "gem install solargraph", Plugin: "solargraph-lsp"}`
- DefaultDomains: `["rubygems.org"]`
- DynamicDomains: `[]string{}`
- MarkerFiles: `["Gemfile"]`

**Rust**
- Runtime: `{Tool: "rust", Version: "latest"}`
- LSP: `{Package: "rust-analyzer", InstallCmd: "rustup component add rust-analyzer", Plugin: "rust-analyzer-lsp"}`
- DefaultDomains: `["crates.io", "static.crates.io", "index.crates.io"]`
- DynamicDomains: `["static.rust-lang.org"]` (Rust CDN uses changing IPs for distribution)
- MarkerFiles: `["Cargo.toml"]`

### Steps

1. **Define types** -- Add `StackID`, `Runtime`, `LSP`, and `Stack` types to `internal/stack/stack.go`. Use a doc comment on the package explaining its role as the single source of truth. Constants for all five `StackID` values go in a const block.

2. **Define the package-level registry** -- Declare an unexported `var registry map[StackID]Stack`. Populate it using a package-level `init()` function with all five stack entries. Using `init()` here is appropriate because this is static data initialization with no side effects (unlike command registration, where we avoid `init()`).

3. **Implement `Get(id StackID) (Stack, bool)`** -- Look up the ID in the map. If found, return a deep copy of the Stack (clone the slice fields: `DefaultDomains`, `DynamicDomains`, `MarkerFiles`). Return `false` if not found. Deep-copying prevents callers from mutating the registry, as required by `TestAll_ReturnsCopies`.

4. **Implement `All() []Stack`** -- Return a slice of deep-copied Stack values, sorted alphabetically by `StackID`. Iterate the map keys, sort them, then build the result slice using the same deep-copy helper. The sort is required by `TestAll_Sorted`.

5. **Implement `IDs() []StackID`** -- Return a sorted slice of all registered `StackID` values. Required by `TestIDs_Sorted` and `TestIDs_MatchesAll`.

6. **Extract a `copyStack` helper** -- Unexported function that deep-copies a `Stack` value (copies the struct and clones all slice fields using `slices.Clone` or manual `append([]string(nil), src...)`). Used by both `Get` and `All` to ensure immutability.

7. **Verify all tests pass** -- Run `go test ./internal/stack/...` and confirm all 9 existing tests pass. Run `go test ./...` to confirm no regressions. Run `golangci-lint run ./...` to confirm clean lint.

### Testing Strategy

The test file already exists at `internal/stack/stack_test.go` with 9 comprehensive tests. No new tests need to be written. The existing tests validate:

- **TestRegistryCompleteness** -- Exactly 5 stacks registered; all expected IDs present.
- **TestGet_ExistingStack** -- Each stack has non-empty ID, Name, Runtime.Tool, Runtime.Version, LSP.Package, LSP.InstallCmd, LSP.Plugin, DefaultDomains, and MarkerFiles.
- **TestGet_UnknownStack** -- Unknown ID returns `ok=false`.
- **TestAll_ReturnsCopies** -- Mutations to returned values do not leak back into the registry.
- **TestAll_Sorted** -- `All()` returns stacks sorted by ID.
- **TestIDs_Sorted** -- `IDs()` returns sorted IDs.
- **TestIDs_MatchesAll** -- `IDs()` and `All()` return the same set of IDs.
- **TestNoDuplicateMarkerFiles** -- No marker file is claimed by two stacks.
- **TestNoDuplicateDomains** -- No duplicate entries within a stack's DefaultDomains or DynamicDomains.
- **TestDomainsAreValidHostnames** -- All domain strings match a hostname regex.

### Design Decisions

**Why `internal/stack/` and not `internal/detect/`?**
The registry is pure data. Detection (scanning files on disk) is behavior. Keeping them separate respects the single-responsibility principle. The `detect` package will import `stack` to look up marker files, and the `render` and `firewall` packages will also import `stack` for domains and runtime info. Placing the registry in `detect` would create an awkward dependency from `render` -> `detect`.

**Why `init()` for population?**
The CLAUDE.md rule about avoiding `init()` applies to Cobra command registration (where it hinders test isolation). A data-only `init()` that populates an immutable map is idiomatic Go for registries and has no testability downside -- the `Get`/`All`/`IDs` functions are stateless reads against static data.

**Why deep copies?**
`TestAll_ReturnsCopies` explicitly tests for this. Slices in Go are reference types; returning them directly would allow callers to corrupt the registry. The `copyStack` helper clones all three slice fields.

**Why `DynamicDomains` is empty for most stacks?**
Only Rust currently has a dynamic domain (`static.rust-lang.org` whose CDN IPs rotate). The field exists on every stack so the merging logic (ccbox-tv1t) and firewall package can handle both domain types uniformly without special-casing.

### Open Questions

None. The test file fully specifies the API contract. The data values (tool names, domains, marker files) are well-established conventions. If domain lists need refinement, they can be adjusted later without API changes.

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-02 |
| challenge | complete | 1 | 2026-04-02 |
| implement | complete | 1 | 2026-04-02 |
| pr | complete | 1 | 2026-04-02 |
| review | complete | 1 | 2026-04-02 |
| rework | complete | 1 | 2026-04-02 |
| codify | complete | 1 | 2026-04-02 |