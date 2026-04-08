---
# agentbox-5x61
title: Install golangci-lint in generated Dockerfile for Go stack
status: in-progress
type: feature
priority: normal
created_at: 2026-04-07T19:30:05Z
updated_at: 2026-04-08T06:43:59Z
---

When Go is a selected stack, the generated Dockerfile should install golangci-lint automatically. This is a standard Go development tool that every Go project needs. Currently it has to be installed manually after container creation and doesn't survive rebuilds.

The Go stack metadata in internal/stack/stack.go should include golangci-lint as a dev tool, and the Dockerfile template should install it (via go install from proxy.golang.org, since GitHub release downloads may be blocked by the firewall).

Also update this project's .devcontainer/ to include it.

## Definition of Done

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [ ] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [x] Branch pushed to remote
- [x] PR created
- [x] Automated code review passed via `@review-backend` subagent (via Task tool)
- [x] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [x] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes) — N/A, no new patterns
- [x] All other checklist items above are completed
- [x] User notified for human review

## Implementation Plan

### Approach

Add a `DevTools` field to the `Stack` struct -- a slice of install commands for per-stack development tools that are not LSPs but should be installed in the generated Dockerfile. This follows the same pattern as `SystemDeps` (slice of strings, collected and deduplicated by `Merge`). The Dockerfile template renders these as individual `RUN` commands in a new "Dev tools" section after LSPs and before Claude Code.

golangci-lint is the first (and currently only) entry, added to the Go stack with the install command `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`. This uses `proxy.golang.org` which is already in Go's allowlisted domains, so no firewall changes are needed.

This approach is chosen over alternatives:
- **Overloading `LSP`**: golangci-lint is not an LSP and does not have a Claude Code plugin -- forcing it into the LSP model would be misleading.
- **Hardcoding in template**: Violates the registry pattern (all stack-specific behavior flows from `internal/stack/` metadata).
- **Single `DevToolCmd string` field**: A slice of structs is more extensible for future tools and consistent with `SystemDeps`.

### Data Model: `DevTool` struct

A lightweight struct rather than a bare string, for clarity and future extensibility:

```go
// DevTool describes a development tool installed in the generated container.
type DevTool struct {
    Name       string // Human-readable name, e.g. "golangci-lint"
    InstallCmd string // Full install command, e.g. "go install .../golangci-lint/v2/cmd/golangci-lint@latest"
}
```

This parallels `LSP` (which also has a `Package` name and `InstallCmd`). Using a struct instead of a bare string allows the Dockerfile template comment to identify what is being installed.

### Files to Create/Modify

1. **`internal/stack/stack.go`** -- Add `DevTool` struct, add `DevTools []DevTool` field to `Stack`, add golangci-lint to Go stack registry entry, clone `DevTools` in `copyStack`.
2. **`internal/stack/stack_test.go`** -- Add tests for `DevTools` field: non-nil invariant, defensive copy, known values spot-check, no duplicates.
3. **`internal/render/render.go`** -- Add `DevTools []stack.DevTool` field to `GenerationConfig`, collect and deduplicate dev tools in `Merge` by `Name`.
4. **`internal/render/render_test.go`** -- Add merge tests for `DevTools`: Go-only has 1 dev tool, non-Go stacks have 0, deduplication across stacks, sorted output, empty/nil slices.
5. **`internal/render/templates/Dockerfile.tmpl`** -- Add `{{ if .DevTools }}` block after LSPs section, before Claude Code section.
6. **`internal/render/dockerfile_test.go`** -- Add tests: golangci-lint appears for Go stack, absent for non-Go stacks, structural test for all stacks, no template artifacts.
7. **`.devcontainer/Dockerfile`** -- Add `RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` to this project's own devcontainer.
8. **`cmd/init_integration_test.go`** -- Add assertion that golangci-lint install command appears in generated Dockerfile for Go stack, and is absent for non-Go stacks.

### Steps

#### Step 1: Add `DevTool` struct and `DevTools` field to stack registry

File: `/workspace/internal/stack/stack.go`

- Define `DevTool` struct with `Name string` and `InstallCmd string` fields. Place it near `LSP` struct.
- Add `DevTools []DevTool` field to `Stack` struct, with doc comment explaining its purpose. Place after `LSP` field.
- Add golangci-lint to the Go registry entry:
  ```go
  DevTools: []DevTool{{
      Name:       "golangci-lint",
      InstallCmd: "go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest",
  }},
  ```
- Add `DevTools: []DevTool{}` (non-nil empty) to Node, Python, Rust, and Ruby registry entries.
- Clone `DevTools` in `copyStack`: `cp.DevTools = slices.Clone(s.DevTools)`.

#### Step 2: Add stack registry tests for DevTools

File: `/workspace/internal/stack/stack_test.go`

- `TestDevTools_NonNil` -- Iterate all stacks, assert `DevTools` is never nil (same pattern as `TestSystemDeps_NonNil`).
- `TestDevTools_DefensiveCopy` -- Get Go stack, append to `DevTools`, get again, verify mutation did not leak (same pattern as `TestSystemDeps_DefensiveCopy`).
- `TestDevTools_KnownValues` -- Get Go stack, assert it contains a `DevTool` with `Name == "golangci-lint"` and `InstallCmd` containing `golangci-lint/v2`. Assert Node stack has empty `DevTools`.
- `TestDevTools_NoDuplicates` -- Iterate all stacks, check no duplicate `Name` values within a single stack's `DevTools`.

#### Step 3: Add `DevTools` to `GenerationConfig` and `Merge`

File: `/workspace/internal/render/render.go`

- Add `DevTools []stack.DevTool` field to `GenerationConfig` struct.
- In `Merge`, after the system deps collection loop (Step 3 in existing code), add a new step to collect dev tools. Deduplicate by `Name` (same pattern as LSPs deduplicate by `Package`):
  ```go
  seenDevTools := make(map[string]bool)
  var devTools []stack.DevTool
  for _, id := range uniqueStacks {
      s, _ := stack.Get(id)
      for _, dt := range s.DevTools {
          if !seenDevTools[dt.Name] {
              seenDevTools[dt.Name] = true
              devTools = append(devTools, dt)
          }
      }
  }
  slices.SortFunc(devTools, func(a, b stack.DevTool) int {
      return strings.Compare(a.Name, b.Name)
  })
  ```
- Add nil-to-empty normalization: `if devTools == nil { devTools = []stack.DevTool{} }`.
- Include `DevTools: devTools` in the returned `GenerationConfig`.

#### Step 4: Add Merge tests for DevTools

File: `/workspace/internal/render/render_test.go`

- `TestMerge_DevTools_GoOnly` -- Merge with Go only, assert `DevTools` has 1 entry with `Name == "golangci-lint"`.
- `TestMerge_DevTools_NonGoStack` -- Merge with Node only, assert `DevTools` is non-nil empty slice.
- `TestMerge_DevTools_MultiStackWithGo` -- Merge with Go + Node, assert `DevTools` has 1 entry (golangci-lint).
- `TestMerge_DevTools_Empty` -- Merge with empty stacks, assert `DevTools` is non-nil empty slice.
- `TestMerge_DevTools_Sorted` -- Merge all stacks, assert `DevTools` is sorted by `Name`.
- `TestMerge_DevTools_Deduplication` -- Merge with Go + Go (duplicate), assert `DevTools` has exactly 1 entry.

#### Step 5: Add DevTools section to Dockerfile template

File: `/workspace/internal/render/templates/Dockerfile.tmpl`

Add a new conditional block after the LSPs section (line 53) and before the Claude Code section (line 54). The block mirrors the LSPs pattern:

```
{{ if .DevTools }}
# --- Dev tools ---
{{ range .DevTools -}}
RUN {{ .InstallCmd }}
{{ end -}}
{{ end -}}
```

This places dev tool installs after LSPs and before Claude Code, which is the right ordering because:
- Dev tools may depend on runtimes (golangci-lint needs Go in PATH), which are installed via `mise install` above.
- They run as the `node` user (same USER context as LSPs).
- They must precede the `USER root` switch that follows Claude Code.

#### Step 6: Add Dockerfile template tests for DevTools

File: `/workspace/internal/render/dockerfile_test.go`

- `TestDockerfile_DevTools_GoStack` -- Merge with Go, render Dockerfile, assert output contains `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`.
- `TestDockerfile_DevTools_AbsentForNonGoStack` -- Merge with Node only, render Dockerfile, assert output does NOT contain `golangci-lint`.
- `TestDockerfile_DevTools_AllStacks` -- Update existing `TestDockerfile_AllStacks` to add structural assertion: every `cfg.DevTools` entry's `InstallCmd` appears in output.
- `TestDockerfile_DevTools_EmptyConfig` -- Verify empty config produces no "Dev tools" section. Update existing `TestDockerfile_EmptyConfig` with negative assertion for `golangci-lint`.
- `TestDockerfile_DevTools_OrderingInDockerfile` -- Assert the golangci-lint install command appears AFTER `mise install` and BEFORE `npm install -g @anthropic-ai/claude-code` in the output.
- Verify existing `TestDockerfile_NoTemplateArtifacts` and `TestDockerfile_Deterministic` continue to pass (they exercise all stacks, so they will cover `DevTools` automatically).

#### Step 7: Update this project's own .devcontainer/Dockerfile

File: `/workspace/.devcontainer/Dockerfile`

Add a new line after the LSP servers section (after line 49, `RUN go install golang.org/x/tools/gopls@latest`):

```dockerfile
# --- Dev tools ---
RUN go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
```

This goes before the `# --- Claude Code ---` section, matching the ordering the template will produce.

#### Step 8: Update integration tests

File: `/workspace/cmd/init_integration_test.go`

- In `TestIntegration_SingleGoStack`: Add assertion that `dockerfile` contains `golangci-lint`.
- In `TestIntegration_MultiStack` (Go + Node): Add assertion that `dockerfile` contains `golangci-lint`.
- Add a new test `TestIntegration_NonGoStack_NoDevTools`: Create a temp dir with only `package.json`, run init with `--stack node`, verify Dockerfile does NOT contain `golangci-lint`.

### Testing Strategy

**Unit tests (TDD -- write these first):**

1. `internal/stack/stack_test.go` -- Registry invariants for the new `DevTools` field: non-nil for all stacks, defensive copy, known values, no duplicates.
2. `internal/render/render_test.go` -- Merge behavior for `DevTools`: collection, deduplication, sorting, empty/nil handling.
3. `internal/render/dockerfile_test.go` -- Template rendering: presence for Go, absence for non-Go, structural completeness for all stacks, ordering within Dockerfile, no template artifacts.

**Integration tests:**

4. `cmd/init_integration_test.go` -- End-to-end: Go stack produces Dockerfile with golangci-lint, non-Go stack does not.

**What to verify manually:**

5. Rebuild this project's own devcontainer after modifying `.devcontainer/Dockerfile` to confirm golangci-lint is available.

### Edge Cases

- **Non-Go stacks**: `DevTools` is empty, template `{{ if .DevTools }}` block is skipped entirely -- no empty "Dev tools" comment in output.
- **Go + Go duplicate input**: Deduplication in `Merge` prevents double golangci-lint install.
- **Future multi-tool stacks**: If a stack adds multiple dev tools, they are sorted by `Name` for deterministic output. The slice-based model supports this without changes.
- **Version pinning**: The install command uses `@latest`. Users who want a pinned version can edit the generated Dockerfile after generation. This is consistent with how LSP install commands work (they all use `@latest`).

### Open Questions

None -- the design is straightforward and follows established patterns exactly.
