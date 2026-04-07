---
# agentbox-dn7n
title: Extract mise config.toml to .devcontainer as standalone file
status: completed
type: feature
priority: normal
created_at: 2026-04-07T17:33:46Z
updated_at: 2026-04-07T19:24:22Z
---

Currently the mise TOML config is generated inline in the Dockerfile via a COPY heredoc. This has two problems: (1) versions are hardcoded (node=lts, everything else=latest) with no way to customize, and (2) the user can't easily edit runtime versions after generation.

**Goal**: Make the mise config.toml a first-class generated file in .devcontainer/, and the single source of truth for runtime versions. The Dockerfile should COPY it in, not generate it.

## Changes Required

### 1. New render function for config.toml
- Add `internal/render/templates/mise-config.toml.tmpl` template
- Add `render.MiseConfig(w io.Writer, cfg GenerationConfig) error` function
- Node is always present (Claude Code dependency) but version comes from config, not hardcoded
- All runtimes rendered from `GenerationConfig.Runtimes` — no special-casing node in template

### 2. Update Dockerfile template
- Remove the inline COPY heredoc that generates config.toml (lines 38-48 of Dockerfile.tmpl)
- Replace with `COPY config.toml /home/node/.config/mise/config.toml`
- No version information in the Dockerfile at all — config.toml is the single source of truth

### 3. Add config.toml to file manifest
- `cmd/init.go`: render and write `.devcontainer/config.toml` alongside existing files

### 4. Wizard version prompting
- Allow users to specify runtime versions in the interactive wizard (default: latest/lts)
- Non-interactive mode: use defaults (latest for most, lts for node)
- Plumb version choices through to `GenerationConfig.Runtimes`

## Out of scope
- Changing stack detection logic
- Changing firewall/domain logic

## Implementation Plan

### Approach

Extract the inline mise TOML heredoc from `Dockerfile.tmpl` into a standalone `mise-config.toml.tmpl` template, rendered by a new `render.MiseConfig` function. The Dockerfile changes to a simple `COPY config.toml ...` line. The `cmd/init.go` file manifest grows by one entry (`config.toml`). The wizard gains a per-runtime version prompting step. A `--runtime-version` CLI flag enables non-interactive/scripted version pinning.

**Key design decision**: The `Merge` function remains pure -- it does NOT inject node into Runtimes. Instead, a new `render.EnsureNode(cfg *GenerationConfig)` helper is called from `cmd/init.go` after `Merge` returns and after version overrides are applied. This keeps `Merge` as a faithful reflection of registry data for the selected stacks, while moving the "node is always present for Claude Code" invariant to an explicit, testable step at the orchestration layer. The Dockerfile template's inline `node = "lts"` hardcoding and `{{ if ne .Tool "node" }}` special-casing are both removed; the new `mise-config.toml.tmpl` iterates ALL runtimes uniformly.

### Files to Create/Modify

1. **`internal/render/templates/mise-config.toml.tmpl`** (CREATE) -- New TOML template for mise config
2. **`internal/render/mise.go`** (CREATE) -- New render function `MiseConfig(w io.Writer, cfg GenerationConfig) error`
3. **`internal/render/mise_test.go`** (CREATE) -- Tests for mise config rendering
4. **`internal/render/ensure.go`** (CREATE) -- `EnsureNode(cfg *GenerationConfig)` helper
5. **`internal/render/ensure_test.go`** (CREATE) -- Tests for EnsureNode
6. **`internal/render/templates/Dockerfile.tmpl`** (MODIFY) -- Remove inline heredoc, replace with `COPY config.toml`
7. **`internal/render/dockerfile.go`** (NO CHANGE) -- The `Dockerfile` render function stays as-is
8. **`internal/render/dockerfile_test.go`** (MODIFY) -- Update tests: remove mise tool content assertions from Dockerfile, add assertion for `COPY config.toml`
9. **`internal/render/render.go`** (NO CHANGE) -- `Merge` remains pure, does not inject node
10. **`internal/render/render_test.go`** (NO CHANGE) -- Existing `Merge` tests remain unchanged (Merge is not modified)
11. **`internal/wizard/wizard.go`** (MODIFY) -- Add per-runtime version prompting to the wizard; expand `Choices` to include `RuntimeVersions`
12. **`internal/wizard/wizard_test.go`** (MODIFY) -- Test `Choices` zero-value with new field, test `buildSummary` with version info
13. **`cmd/init.go`** (MODIFY) -- Call `EnsureNode`, render `config.toml`, add to file manifest, add `--runtime-version` flag, apply version overrides
14. **`cmd/init_test.go`** (MODIFY) -- Update `expectedFiles`, update `fakePrompter`, add version override tests, add `--runtime-version` flag tests
15. **`cmd/init_integration_test.go`** (MODIFY) -- Add `config.toml` to `expectedFiles`, update content assertions

### Steps

#### Step 1: Create `EnsureNode` helper in `internal/render/`

**File**: `internal/render/ensure.go` (CREATE)

Add a new function that ensures node is present in `GenerationConfig.Runtimes`:

```go
package render

import (
    "slices"
    "strings"

    "github.com/bjro/agentbox/internal/stack"
)

// EnsureNode guarantees that the "node" runtime is present in cfg.Runtimes.
// Claude Code requires npm, so node must always be installed via mise.
// If node is already present (e.g., because the user selected the Node stack),
// this function is a no-op. Otherwise, it appends node with version "lts" and
// re-sorts to maintain the Tool-sorted invariant.
//
// This is intentionally separate from Merge to keep Merge as a pure reflection
// of registry data for the selected stacks.
func EnsureNode(cfg *GenerationConfig) {
    for _, r := range cfg.Runtimes {
        if r.Tool == "node" {
            return
        }
    }
    cfg.Runtimes = append(cfg.Runtimes, stack.Runtime{Tool: "node", Version: "lts"})
    slices.SortFunc(cfg.Runtimes, func(a, b stack.Runtime) int {
        return strings.Compare(a.Tool, b.Tool)
    })
}
```

Key design points:
- Takes `*GenerationConfig` (pointer) since it mutates Runtimes in-place.
- Uses the same sort comparator as `Merge` to maintain the sorted invariant.
- No-op when node is already present (idempotent).
- The "lts" default matches `stack.Node.Runtime.Version` in the registry.

**File**: `internal/render/ensure_test.go` (CREATE)

Tests for the `EnsureNode` helper:

- `TestEnsureNode_InjectsWhenMissing` -- Start with Go-only Runtimes `[{Tool: "go", Version: "latest"}]`. After `EnsureNode`, verify node is present with `Version == "lts"` and total length is 2.
- `TestEnsureNode_NoOpWhenPresent` -- Start with Node in Runtimes `[{Tool: "node", Version: "20"}]`. After `EnsureNode`, verify node version is still "20" (not overwritten to "lts") and total length is still 1.
- `TestEnsureNode_PreservesExistingNodeVersion` -- Start with Go + Node (custom version "18") in Runtimes. After `EnsureNode`, verify node version is still "18".
- `TestEnsureNode_MaintainsSortOrder` -- Start with `[{Tool: "ruby", Version: "latest"}, {Tool: "go", Version: "latest"}]` (unsorted is fine, but `Merge` produces sorted). After `EnsureNode`, verify Runtimes are sorted by Tool: go, node, ruby.
- `TestEnsureNode_EmptyRuntimes` -- Start with `[]stack.Runtime{}`. After `EnsureNode`, verify single entry `{Tool: "node", Version: "lts"}`.
- `TestEnsureNode_NilRuntimes` -- Start with nil Runtimes. After `EnsureNode`, verify single entry `{Tool: "node", Version: "lts"}` and slice is non-nil.
- `TestEnsureNode_Idempotent` -- Call `EnsureNode` twice. Verify result is identical after both calls.

#### Step 2: Create mise-config.toml template and render function

**File**: `internal/render/templates/mise-config.toml.tmpl` (CREATE)

```toml
[tools]
{{ range .Runtimes -}}
{{ .Tool }} = "{{ .Version }}"
{{ end -}}
```

Simple iteration over all runtimes. No node special-casing -- node is guaranteed present in `.Runtimes` by `EnsureNode` called from `cmd/init.go`.

**File**: `internal/render/mise.go` (CREATE)

New file following the `devcontainer.go` pattern (`template.ParseFS` from shared `templateFS`):

```go
package render

import (
    "fmt"
    "io"
    "text/template"
)

var miseConfigTmpl = template.Must(template.ParseFS(templateFS, "templates/mise-config.toml.tmpl"))

// MiseConfig renders the mise config.toml template to w. The output is a TOML
// configuration file for the mise runtime manager, listing all runtime tools
// and their versions. Node is expected to be present in cfg.Runtimes (ensured
// by EnsureNode at the call site).
func MiseConfig(w io.Writer, cfg GenerationConfig) error {
    if err := miseConfigTmpl.ExecuteTemplate(w, "mise-config.toml.tmpl", cfg); err != nil {
        return fmt.Errorf("render mise config.toml: %w", err)
    }
    return nil
}
```

Note: uses `ExecuteTemplate` with the template name (like `devcontainer.go` pattern) rather than `Execute`, because `template.ParseFS` names templates by their filename.

**File**: `internal/render/mise_test.go` (CREATE)

Tests following the two-tier strategy:

- `TestMiseConfig_SingleStack` -- Build config via Merge for Go only, call `EnsureNode`, render: output contains `go = "latest"` and `node = "lts"`.
- `TestMiseConfig_MultiStack` -- Build config via Merge for Go + Python, call `EnsureNode`, render: contains go, python, and node entries.
- `TestMiseConfig_AllStacks` -- All five stacks via Merge + `EnsureNode`: structural assertion that every runtime in `cfg.Runtimes` appears in output. Node appears exactly once.
- `TestMiseConfig_DirectConfig_CustomVersions` -- Hand-built config with `{Tool: "go", Version: "1.22.0"}, {Tool: "node", Version: "20"}`: verifies the exact version strings appear.
- `TestMiseConfig_DirectConfig_EmptyRuntimes` -- Empty (non-nil) runtimes: output contains `[tools]` header and no `<no value>` artifacts.
- `TestMiseConfig_NoTemplateArtifacts` -- All-stacks config with `EnsureNode`: no `<no value>`, `<nil>`, `{{`, `}}` in output.
- `TestMiseConfig_Deterministic` -- Two renders produce identical bytes.
- `TestMiseConfig_TOMLFormat` -- Verify output starts with `[tools]` and non-blank, non-header lines match `tool = "version"` format.
- `TestMiseConfig_TrailingNewline` -- Verify output ends with a single trailing newline (POSIX text file convention). Assert `strings.HasSuffix(out, "\n")` and `!strings.HasSuffix(out, "\n\n")`.
- `TestMiseConfig_NoTrailingWhitespace` -- Verify no line has trailing spaces or tabs, consistent with Dockerfile whitespace test.

#### Step 3: Update Dockerfile template

**File**: `internal/render/templates/Dockerfile.tmpl` (MODIFY)

Replace lines 38-48 (the `# --- Runtime configuration ---` comment through the `MISE` heredoc delimiter) with:

```dockerfile
# --- Runtime configuration ---
# Managed by mise; edit .devcontainer/config.toml to change versions.
COPY config.toml /home/node/.config/mise/config.toml
```

This removes the inline COPY heredoc, the `node = "lts"` hardcoding, and the `{{ range .Runtimes }}` / `{{ if ne .Tool "node" }}` loop. The Dockerfile template becomes fully static with respect to runtime tools (the only remaining `{{ range }}` actions are for `.SystemDeps` and `.LSPs`). The rest of the Dockerfile (chown, USER node, mise install, LSP installs, etc.) remains unchanged.

**File**: `internal/render/dockerfile_test.go` (MODIFY)

Update tests to reflect that the Dockerfile no longer contains mise tool version entries:

- `TestDockerfile_MiseToolsSingleStack`: Remove assertions for `go = "latest"` and `node = "lts"` in Dockerfile. Add assertion for `COPY config.toml /home/node/.config/mise/config.toml`. Rename to `TestDockerfile_MiseConfigCopied_SingleStack`.
- `TestDockerfile_MiseToolsMultiStack`: Remove all `tool = "version"` assertions. Add COPY assertion. Rename to `TestDockerfile_MiseConfigCopied_MultiStack`.
- `TestDockerfile_EmptyConfig`: Remove `node = "lts"` assertion and non-node tool absence assertions. Add COPY assertion. Keep Claude Code install assertion.
- `TestDockerfile_DirectConfig_CustomRuntimesAndLSPs`: Remove `deno = "1.40"` and `zig = "0.12"` assertions (runtime versions are no longer in Dockerfile). Keep LSP and system dep assertions. Add COPY assertion.
- `TestDockerfile_AllStacks`: Remove the mise runtime entry loop (lines 513-520). Remove node count assertion (lines 539-542). Add COPY assertion. Keep LSP and system dep assertions.
- `TestDockerfile_NodeAlwaysInMiseConfig`: Rename to `TestDockerfile_MiseConfigCopied` and simplify to verifying `COPY config.toml /home/node/.config/mise/config.toml` exists for all stack combos. The node-always-present invariant is now tested in `ensure_test.go` and `mise_test.go`.
- Add whitespace assertion to `TestDockerfile_NoTrailingWhitespace`: explicitly assert no trailing newline doubling (i.e., file ends with exactly one newline).

#### Step 4: Add version prompting to wizard and `--runtime-version` CLI flag

**File**: `internal/wizard/wizard.go` (MODIFY)

Expand `Choices` struct:

```go
type Choices struct {
    Stacks          []stack.StackID
    ExtraDomains    []string
    RuntimeVersions map[string]string // tool name -> version (empty/nil map means use defaults)
}
```

In `HuhPrompter.Run`, after Form 1 (stack selection + extra domains) and before Form 2 (confirmation), add a Form 1.5 for runtime version prompting:

- Collect all runtime tools for the selected stacks by looking up each stack in the registry. Always include node (with default `"lts"`) even if not in selected stacks, since `EnsureNode` will add it.
- Sort the tools alphabetically for deterministic form ordering.
- Create one `huh.NewInput()` per runtime tool, with the tool name as title, the registry default version as the initial/placeholder value, and a description: `"Press Enter to accept default"`.
- This addresses the UX concern from the challenge: users see clear guidance that they can press Enter to keep defaults.
- Store the results in a `map[string]string`. Only include entries where the user typed a value different from the default (or include all -- both approaches are valid since overriding with the same value is a no-op).

Update `buildSummary` to accept a third parameter `runtimeVersions map[string]string` and show runtime versions in the confirmation summary. The signature becomes:

```go
func buildSummary(stacks []stack.StackID, extraDomains []string, runtimeVersions map[string]string) string
```

Add a "Runtimes" section to the summary output (e.g., `"Runtimes: go=latest, node=lts"`).

**File**: `internal/wizard/wizard_test.go` (MODIFY)

- Update `TestChoices_ZeroValue`: verify `RuntimeVersions` is nil on zero-value `Choices`.
- Update `TestPrompterInterface_FakeImplementation`: add `RuntimeVersions: map[string]string{"go": "1.22"}` to the fake's canned choices and verify round-trip.
- Update `TestBuildSummary_StacksOnly`: update call signature to pass `nil` for runtime versions. Verify summary does NOT contain "Runtimes" line when versions are nil.
- Update `TestBuildSummary_WithDomains`: update call signature to pass `nil` for runtime versions.
- Update `TestBuildSummary_NoDomains`: update call signature to pass `nil` for runtime versions.
- Add `TestBuildSummary_WithRuntimeVersions`: call `buildSummary` with `map[string]string{"go": "1.22", "node": "20"}`. Verify summary contains "Runtimes:" line with "go=1.22" and "node=20".
- Add `TestBuildSummary_EmptyRuntimeVersions`: call with `map[string]string{}`. Verify summary does NOT contain "Runtimes" line (same as nil).

#### Step 5: Wire everything in `cmd/init.go`

**File**: `cmd/init.go` (MODIFY)

Add a `--runtime-version` flag:

```go
var runtimeVersions []string
cmd.Flags().StringSliceVar(&runtimeVersions, "runtime-version", nil,
    "Runtime version overrides as tool=version pairs (e.g., go=1.22,node=20)")
```

Restructure the RunE to:
1. Hoist `choices` variable (type `wizard.Choices`) to function scope (declare before the `stackFlagSet` branch) so it is accessible after the wizard/non-interactive branch.
2. After `render.Merge`, call `render.EnsureNode(&cfg)` to guarantee node is present.
3. Apply version overrides using merge semantics (Finding 3: Option A):

```go
// Apply version overrides: initialize from wizard, then layer CLI flags on top.
versionOverrides := make(map[string]string)
// Wizard overrides (from interactive prompting).
for k, v := range choices.RuntimeVersions {
    versionOverrides[k] = v
}
// CLI flag overrides take precedence (for scripted use).
if len(runtimeVersions) > 0 {
    runtimeVersions = trimAndFilter(runtimeVersions)
    parsed, parseErr := parseRuntimeVersions(runtimeVersions)
    if parseErr != nil {
        return parseErr
    }
    for k, v := range parsed {
        versionOverrides[k] = v
    }
}
// Apply to cfg.Runtimes.
for i, rt := range cfg.Runtimes {
    if v, ok := versionOverrides[rt.Tool]; ok && v != "" {
        cfg.Runtimes[i].Version = v
    }
}
```

This merge approach means: if the wizard sets `go=1.21` and the CLI flag sets `go=1.22`, the CLI flag wins. If the wizard sets `node=18` and the CLI flag does not mention node, the wizard's `node=18` is preserved.

4. Render `config.toml` and add to file manifest:

```go
var miseConfigBuf bytes.Buffer
if err := render.MiseConfig(&miseConfigBuf, cfg); err != nil {
    return err
}
```

Add `"config.toml": miseConfigBuf.Bytes()` to the `files` map.

5. Add a `parseRuntimeVersions` helper function:

```go
func parseRuntimeVersions(pairs []string) (map[string]string, error) {
    result := make(map[string]string, len(pairs))
    for _, pair := range pairs {
        parts := strings.SplitN(pair, "=", 2)
        if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
            return nil, fmt.Errorf("invalid --runtime-version %q; expected tool=version format (e.g., go=1.22)", pair)
        }
        tool := strings.TrimSpace(parts[0])
        version := strings.TrimSpace(parts[1])
        result[tool] = version
    }
    return result, nil
}
```

No validation that the tool name matches a known runtime. Unknown tools are silently ignored (no runtime to override). This keeps the flag simple and avoids coupling to the registry.

**File**: `cmd/init_test.go` (MODIFY)

- Update `fakePrompter.choices` in `TestInitCommand_WizardFlow` to include `RuntimeVersions: nil` (explicit for clarity).
- Update all `expected` file lists to include `"config.toml"` (now 9 files).
- Add `TestInitCommand_RuntimeVersionFlag`: test `--runtime-version go=1.22,node=20 --stack go` and verify `config.toml` contains `go = "1.22"` and `node = "20"`.
- Add `TestInitCommand_RuntimeVersionFlagInvalid`: test malformed `--runtime-version` values (e.g., `"golatest"`, `"=1.22"`, `"go="`) produce clear errors.
- Add `TestInitCommand_WizardVersionOverrides`: fake prompter returns custom `RuntimeVersions: map[string]string{"go": "1.21"}`, verify `config.toml` contains `go = "1.21"`.
- Add `TestInitCommand_WizardAndFlagMerge`: fake prompter returns `RuntimeVersions: map[string]string{"go": "1.21", "node": "18"}`, also pass `--runtime-version node=20`. Verify `config.toml` contains `go = "1.21"` (from wizard) and `node = "20"` (CLI overrides wizard).
- Add `TestParseRuntimeVersions`: unit tests for the `parseRuntimeVersions` helper (valid pairs, empty tool, empty version, missing `=`, whitespace handling).
- Add `TestInitCommand_ConfigTomlExists`: verify `config.toml` file exists and is non-empty in a basic init run.

#### Step 6: Update integration tests

**File**: `cmd/init_integration_test.go` (MODIFY)

- Update `expectedFiles` slice: add `"config.toml"` (now 9 files).
- `TestIntegration_SingleGoStack`:
  - Add content assertion that `config.toml` contains `go = "latest"` and `node = "lts"`.
  - Replace Dockerfile assertion for `go = "latest"` (line 86-87) with assertion for `COPY config.toml /home/node/.config/mise/config.toml`.
  - Replace negative Dockerfile runtime assertions (lines 93-97) with negative `config.toml` assertions: `config.toml` should not contain `python = "`, `ruby = "`, `rust = "`.
  - Add `config.toml` trailing newline assertion.
- `TestIntegration_MultiStack`:
  - Replace Dockerfile assertion for `go = "latest"` (line 195-196) with COPY assertion.
  - Add assertions that `config.toml` contains `go = "latest"` and `node = "lts"`.
  - Keep LSP install command assertions in Dockerfile (those are unaffected).
- Add `TestIntegration_RuntimeVersionFlag`: run with `--stack go --runtime-version go=1.22,node=20` and verify `config.toml` contains `go = "1.22"` and `node = "20"` instead of defaults. Also verify Dockerfile does NOT contain any `= "1.22"` strings (versions live only in config.toml).

### Testing Strategy

1. **Unit tests in `internal/render/ensure_test.go`**: Verify `EnsureNode` helper: injection when missing, no-op when present, preserves custom versions, maintains sort order, handles nil/empty slices, idempotent.

2. **Unit tests in `internal/render/mise_test.go`**: Structural assertions on template output (TOML format, tool entries, no artifacts, determinism, trailing newline, no trailing whitespace). Both isolation tests (hand-built `GenerationConfig`) and integration tests (through `Merge` + `EnsureNode`).

3. **Unit tests in `internal/render/dockerfile_test.go`**: Verify Dockerfile now contains `COPY config.toml` instead of inline mise TOML. Remove tests that checked for specific runtime versions in Dockerfile.

4. **Unit tests in `internal/wizard/wizard_test.go`**: Verify `Choices.RuntimeVersions` field behavior. Test `buildSummary` with and without version info. Ensure existing tests pass with updated call signature.

5. **Unit tests in `cmd/init_test.go`**: Verify `config.toml` is generated. Test `--runtime-version` flag parsing and validation. Test wizard version overrides flow. Test merge semantics (wizard + CLI flag interaction).

6. **Integration tests in `cmd/init_integration_test.go`**: End-to-end verification that `config.toml` exists with correct content, Dockerfile no longer contains inline mise config, and `--runtime-version` flag works.

7. **Whitespace/newline assertions** (Finding 4): All new template tests include explicit trailing newline assertions (`strings.HasSuffix(out, "\n")` and `!strings.HasSuffix(out, "\n\n")`) and no-trailing-whitespace-per-line assertions.

### Decisions

1. **`Merge` stays pure** (Challenge Finding 1, Option A): `Merge` does not inject node. A separate `EnsureNode` helper called from `cmd/init.go` handles the "node is always present" invariant. This keeps `Merge` as a faithful data-layer merge and makes the node injection testable in isolation.
2. **Wizard version prompting kept** (Challenge Finding 2): The user explicitly requested wizard version prompting. The UX concern is addressed by showing "Press Enter to accept default" in each input's description.
3. **Merge semantics for overrides** (Challenge Finding 3, Option A): Version overrides from wizard and CLI flag are merged, not replaced. Initialize from wizard, then apply CLI flags on top. CLI flags take precedence for overlapping keys.
4. **`buildSummary` signature change** (Challenge Finding 2): Adding `runtimeVersions map[string]string` as a third parameter. All existing call sites (production and test) are updated.
5. **Whitespace assertions** (Challenge Finding 4): All new template tests include explicit trailing newline and no-trailing-whitespace assertions.
6. **File naming**: `config.toml` (matches mise convention).
7. **No `.agentbox.yml` changes**: The generated `.devcontainer/config.toml` is the single source of truth for runtime versions.

### Open Questions

None -- all design decisions have been made.

## Checklist
- [x] Create `EnsureNode` helper in `internal/render/ensure.go` with tests
- [x] Add mise-config.toml.tmpl template
- [x] Add render.MiseConfig function with tests
- [x] Update Dockerfile.tmpl to COPY config.toml instead of inline heredoc
- [x] Update Dockerfile render tests
- [x] Add version prompting to interactive wizard
- [x] Update `buildSummary` signature and tests
- [x] Add `--runtime-version` CLI flag to `agentbox init`
- [x] Add `parseRuntimeVersions` helper with tests
- [x] Wire `EnsureNode`, version overrides, and config.toml rendering in `cmd/init.go`
- [x] Add config.toml to cmd/init.go file manifest
- [x] Update cmd/init_test.go (expectedFiles, fakePrompter, flag tests, merge tests)
- [x] Update integration tests
- [x] Verify existing tests pass

## Definition of Done

- [x] Tests written (TDD: write tests before implementation)
- [x] No new TODO/FIXME/HACK/XXX comments introduced
- [x] `golangci-lint run ./...` passes with no errors
- [x] `go test ./...` passes with no failures
- [x] Branch pushed to remote
- [x] PR created
- [x] Automated code review passed via `@review-backend` subagent (via Task tool)
- [x] Review feedback worked in via `/rework` and pushed to remote (if applicable)
- [ ] ADR written via `/decision` skill (if new dependencies, patterns, or architectural changes)
- [ ] All other checklist items above are completed
- [ ] User notified for human review
