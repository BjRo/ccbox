---
# ccbox-mp0k
title: Generate .ccbox.yml config file
status: in-progress
type: task
priority: low
created_at: 2026-04-02T10:34:46Z
updated_at: 2026-04-03T07:23:53Z
parent: ccbox-puuq
---

## Description
After generation, write a `.ccbox.yml` file to the project root recording the choices made:

```yaml
version: 1
stacks:
  - go
  - node
extra_domains:
  - api.example.com
generated_at: "2026-04-02T10:00:00Z"
ccbox_version: "0.1.0"
```

This serves as documentation of what was generated and could enable a future `ccbox regenerate` command.

## Implementation Plan

### Approach

Add a `Config` struct and marshaling logic to `internal/config/config.go` (currently just a package doc comment), then call it from `cmd/init.go` after writing the `.devcontainer/` files. The config file is pure YAML with five top-level keys. Rather than adding a `gopkg.in/yaml.v3` dependency for this simple structure, use `encoding/json` internally with a hand-written YAML serializer. However, this is a false economy -- the YAML output needs to be human-readable, handle edge cases (empty lists, quoting timestamps), and a future `ccbox regenerate` will need to *read* the file back. Therefore, add `gopkg.in/yaml.v3` as a new dependency. This is the standard Go YAML library and a reasonable addition for a CLI tool that needs to read/write YAML config.

The `internal/config` package will own both the struct definition and the `Write` function. The `cmd/init.go` orchestrator will construct the config struct from data it already has (stacks, extra domains, version, current time) and call `config.Write`. The config package has no dependency on `internal/render` or `internal/stack` -- it receives plain data types (string slices, time.Time, strings) to stay decoupled.

### Files to Create/Modify

- `internal/config/config.go` -- Replace the placeholder package doc with the `Config` struct, `Write` function, and `Load` function
- `internal/config/config_test.go` -- Tests for marshaling, round-tripping, edge cases
- `cmd/init.go` -- After writing `.devcontainer/` files, construct a `config.Config` and call `config.Write` to write `.ccbox.yml` to the project root
- `cmd/init_test.go` -- Add assertions that `.ccbox.yml` is generated with correct content
- `go.mod` / `go.sum` -- Updated by `go get gopkg.in/yaml.v3`

### Steps

1. **Add `gopkg.in/yaml.v3` dependency**
   - Run `go get gopkg.in/yaml.v3`
   - This is the only new dependency introduced

2. **Define `Config` struct in `internal/config/config.go`**
   - Replace the current placeholder file (just a package comment) with the full implementation
   - Define the struct:
     ```
     type Config struct {
         Version      int       `yaml:"version"`
         Stacks       []string  `yaml:"stacks"`
         ExtraDomains []string  `yaml:"extra_domains"`
         GeneratedAt  time.Time `yaml:"generated_at"`
         CcboxVersion string    `yaml:"ccbox_version"`
     }
     ```
   - Use `[]string` for Stacks rather than `[]stack.StackID` to keep the config package decoupled from the stack package. The `cmd` layer converts `stack.StackID` to `string` when constructing the config.
   - `Version` is the schema version (always `1` for now), not the ccbox binary version. This field enables future schema migrations.
   - `ExtraDomains` stores only user-provided extra domains, not the full merged domain list. The full domain list is deterministic from stacks and can be re-derived.
   - Use `flow` style for empty slices to get `extra_domains: []` instead of `extra_domains: null`. This is handled by yaml.v3 marshaling with non-nil empty slices (same pattern used throughout the codebase for template rendering).

3. **Implement `Write` function**
   - Signature: `func Write(w io.Writer, cfg Config) error`
   - Uses `yaml.NewEncoder(w)` to write the config
   - Set `encoder.SetIndent(2)` for consistent formatting
   - Follows the render package pattern of writing to `io.Writer` for testability
   - Returns wrapped error on failure
   - Ensure `Stacks` and `ExtraDomains` are non-nil before marshaling (defensive, same pattern as render package) so YAML renders `[]` not `null` for empty lists

4. **Implement `Load` function**
   - Signature: `func Load(r io.Reader) (Config, error)`
   - Uses `yaml.NewDecoder(r)` to parse the config
   - Validates `Version` field is `1` (return error for unknown versions)
   - Ensures non-nil empty slices after unmarshal for consistent behavior
   - This function enables a future `ccbox regenerate` command and makes round-trip testing possible now

5. **Implement `Filename` constant**
   - `const Filename = ".ccbox.yml"` -- single source of truth for the config filename, used by both `cmd/init.go` and future commands

6. **Write `internal/config/config_test.go`**
   - **Round-trip test**: Create a `Config`, write it, read it back, verify all fields match
   - **YAML format test**: Write a config and verify the output string contains expected YAML structure (e.g., `version: 1`, `stacks:\n- go\n- node`, `extra_domains: []`)
   - **Empty stacks test**: Verify empty stacks renders as `stacks: []` not `stacks: null`
   - **Empty extra domains test**: Verify empty extra_domains renders as `extra_domains: []` not `extra_domains: null`
   - **Load validates version**: Write a config with `version: 99`, attempt to load, verify error
   - **Load non-nil slices**: Load config that omits stacks/extra_domains entirely, verify they are non-nil empty slices
   - **Timestamp round-trip**: Verify the generated_at timestamp survives a write/read cycle with full precision (time.Time marshaled as RFC 3339 by yaml.v3)

7. **Integrate into `cmd/init.go`**
   - After the existing file-writing loop and chmod section, add:
     ```
     ccboxCfg := config.Config{
         Version:      1,
         Stacks:       make([]string, len(stackIDs)),
         ExtraDomains: domains,
         GeneratedAt:  time.Now().UTC(),
         CcboxVersion: version,
     }
     for i, id := range cfg.Stacks {
         ccboxCfg.Stacks[i] = string(id)
     }
     ```
   - Use `cfg.Stacks` (from `render.Merge`) rather than the raw `stackIDs` input, because `cfg.Stacks` is deduplicated and sorted -- the canonical form.
   - For `ExtraDomains`, use the raw `domains` flag value (not the merged domain list). If `domains` is nil (flag not provided), set it to `[]string{}` for clean YAML output.
   - Write the file:
     ```
     ccboxFile, err := os.Create(filepath.Join(dir, config.Filename))
     if err != nil {
         return fmt.Errorf("create %s: %w", config.Filename, err)
     }
     defer ccboxFile.Close()
     if err := config.Write(ccboxFile, ccboxCfg); err != nil {
         return fmt.Errorf("write %s: %w", config.Filename, err)
     }
     ```
   - The `.ccbox.yml` file is written to the project root (`dir`), not inside `.devcontainer/`. This is intentional: it is a project-level config file, like `.gitignore` or `.editorconfig`.
   - Add `"github.com/bjro/ccbox/internal/config"` and `"time"` to imports.

8. **Update `cmd/init_test.go`**
   - In `TestInitCommand_GeneratesDevcontainer`: add assertion that `.ccbox.yml` exists in the project root (not in `.devcontainer/`)
   - Add new test `TestInitCommand_CcboxYmlContent`: run init with `--stacks go,node --domains api.example.com`, then read `.ccbox.yml`, parse it with `config.Load`, and verify:
     - `Version` is 1
     - `Stacks` contains `["go", "node"]`
     - `ExtraDomains` contains `["api.example.com"]`
     - `GeneratedAt` is recent (within last minute)
     - `CcboxVersion` equals the package-level `version` variable
   - Add test `TestInitCommand_CcboxYmlEmptyDomains`: run init with just `--stacks go` (no --domains), verify `ExtraDomains` is empty slice (not nil)

### Design Decisions

- **Why `io.Writer`/`io.Reader` signatures**: Matches the existing `render.DevContainer(w io.Writer, ...)` pattern. Enables testing without filesystem I/O. The `cmd` layer is responsible for creating the actual file.
- **Why plain `[]string` in Config, not `[]stack.StackID`**: Keeps `internal/config` decoupled from `internal/stack`. The config package is a serialization boundary -- it should work with primitive types. The `cmd` layer does the type conversion.
- **Why `gopkg.in/yaml.v3` instead of hand-rolled YAML**: The config will need to be read back for a future `ccbox regenerate` command. Writing a correct YAML parser is not worth the effort. yaml.v3 is the de facto standard Go YAML library. This warrants a brief note in the commit message but not a full ADR since adding a well-known serialization library is not an architectural decision.
- **Why store `extra_domains` not the full domain list**: The full domain list is deterministic given the stacks (derived from the registry). Storing only user extras avoids the config file becoming stale if the registry evolves. A future `regenerate` command re-derives the full list from stacks + extras.
- **Why `version: 1` schema version**: Forward-compatible design. If the config schema changes, bumping the version lets the tool detect old configs and either migrate or error clearly.
- **No ADR needed**: This change adds a YAML config file and a well-known dependency. It does not introduce new architectural patterns or make decisions that constrain future work.

### Testing Strategy

- **Unit tests in `internal/config/config_test.go`**: Test the `Write` and `Load` functions in isolation with `bytes.Buffer`, no filesystem.
- **Integration tests in `cmd/init_test.go`**: Test that `ccbox init` produces a valid `.ccbox.yml` file with correct content.
- **Structural assertions**: Verify YAML structure matches expected format, not exact string matching (timestamps will vary).
- **Round-trip verification**: Write then Load, verify all fields survive the cycle.
- **Edge cases**: Empty stacks, empty domains, nil domains flag.

### Open Questions

None. The bean description is clear and all necessary patterns exist in the codebase.

## Checklist

- [x] Tests written (TDD)
- [x] No TODO/FIXME/HACK/XXX comments
- [x] Lint passes
- [x] Tests pass
- [x] Branch pushed
- [x] PR created
- [x] Automated code review passed
- [x] Review feedback worked in
- [x] ADR written (if architectural changes)
- [ ] User notified

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | done | 1 | 2026-04-03 |
| challenge | done | 1 | 2026-04-03 |
| implement | done | 1 | 2026-04-03 |
| pr | done | 1 | 2026-04-03 |
| review | done | 2 | 2026-04-03 |
| codify | in-progress | 1 | 2026-04-03 |

## Challenge Report

**Scope: SMALL CHANGE** (5 files modified/created, but the logic surface area is small -- struct definition, two functions, and wiring)

### Scope Assessment

| Metric | Value | Threshold |
|--------|-------|-----------|
| Files | 5 (+ go.mod/go.sum) | >15 = recommend split |

### Findings

#### Go Engineer

> **Finding 1: `defer ccboxFile.Close()` swallows the write error on buffered flush** (severity: WARNING)
>
> Step 7 uses `defer ccboxFile.Close()` after `config.Write`. The `yaml.Encoder` from yaml.v3 requires an explicit `Encoder.Close()` call to flush its internal buffer -- the plan delegates this to `config.Write` via `encoder.Encode` which is fine. However, the *file* close via `defer` means the `RunE` function returns `nil` while `os.File.Close()` may still fail (e.g., filesystem full, NFS errors). Since this is the last write operation in the function, the deferred close error is silently dropped.
>
> More importantly, `yaml.NewEncoder` writes to the `io.Writer` during `Encode`, so the data reaches the OS file buffer at that point. But the `os.File.Close()` call is what actually syncs to disk on some filesystems. If it fails, the user thinks the file was written successfully.
>
> This is consistent with how the rest of `cmd/init.go` works (it uses `os.WriteFile` for the devcontainer files, which handles close internally), so the `defer Close` pattern is a divergence from the existing style.
>
> **Option A (recommended):** Use `os.WriteFile` for consistency with the existing pattern. Render to a `bytes.Buffer` first, then write with `os.WriteFile`. This matches lines 97-102 of `cmd/init.go` exactly and avoids the `defer Close` issue entirely. The config YAML is tiny (under 200 bytes), so buffering is free.
> **Option B:** Keep `os.Create` + `defer Close`, but check the close error explicitly by assigning it to a named return and using a deferred closure. More complex than needed for this case.

> **Finding 2: `yaml.v3` renders `time.Time` as `!!timestamp` without quotes by default** (severity: WARNING)
>
> Step 2 shows the struct tag `yaml:"generated_at"` on a `time.Time` field. The bean description example shows the timestamp quoted: `generated_at: "2026-04-02T10:00:00Z"`. However, yaml.v3 marshals `time.Time` as an unquoted YAML timestamp by default (e.g., `generated_at: 2026-04-02T10:00:00Z`). This is valid YAML and will round-trip correctly through yaml.v3's decoder, so it is functionally correct.
>
> The risk is subtle: if a user hand-edits the file and wraps the timestamp in quotes (as shown in the bean description), or if a non-Go YAML parser reads it, the behavior may differ. More practically, the plan's Step 6 test "YAML format test" may assert the wrong expected format if the implementer follows the bean description example literally.
>
> **Option A (recommended):** Accept yaml.v3's default unquoted timestamp output. It is valid YAML 1.1 and round-trips correctly. Note in the implementation that the test should assert the unquoted form, not the quoted form from the bean description. No struct change needed.
> **Option B:** Force quoting by changing the field type to `string` and converting `time.Time` to RFC 3339 string before marshaling / parsing after unmarshaling. This adds complexity for marginal benefit.

### Verdict

APPROVED

The plan is solid. The architecture decisions (decoupled config package, `[]string` over `[]stack.StackID`, `io.Writer`/`io.Reader` signatures, storing only extra domains) are all well-reasoned and consistent with existing codebase patterns. The testing strategy is thorough with good edge case coverage. The two findings above are minor correctness/consistency issues that the implementer should be aware of but do not require re-planning.
