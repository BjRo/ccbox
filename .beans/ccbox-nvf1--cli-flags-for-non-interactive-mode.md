---
# ccbox-nvf1
title: CLI flags for non-interactive mode
status: completed
type: task
priority: normal
created_at: 2026-04-02T10:34:41Z
updated_at: 2026-04-03T08:30:30Z
parent: ccbox-puuq
---

## Description
All wizard options must be expressible as CLI flags for scripting:

```
ccbox init \
  --stack go,node \
  --extra-domains "api.example.com,cdn.example.com" \
  --non-interactive
```

Flags:
- `--stack <comma-separated>`: Explicitly set stacks (skip detection)
- `--extra-domains <comma-separated>`: Additional domains for firewall allowlist
- `--non-interactive` / `-y`: Skip all prompts, use detected stacks + defaults
- `--dir <path>`: Target directory (default: current directory)

When flags provide all required info, skip the wizard entirely.

## Implementation Plan

### Approach

Rename the existing `--stacks` and `--domains` flags to `--stack` and `--extra-domains` to match the bean spec and standard CLI conventions (singular noun for flags that accept comma-separated values). Add two new flags: `--non-interactive`/`-y` (boolean) and `--dir` (string). Refactor the `RunE` function to use `--dir` instead of `os.Getwd()`, validate stack IDs from the flag against the registry, and skip the wizard when sufficient information is provided via flags.

The interactive wizard (ccbox-ogj2) does not exist yet, so the current command is already fully non-interactive. The `--non-interactive` flag is added now to establish the contract: when the wizard bean is implemented later, the wizard should be the default behavior, and `--non-interactive`/`-y` should bypass it. For now, the flag is accepted but the behavior is identical since there is no wizard to skip.

### Flag Name Reconciliation

The existing code has:
- `--stacks` (StringSlice) -> rename to `--stack` (StringSlice)
- `--domains` (StringSlice) -> rename to `--extra-domains` (StringSlice)

Rationale for singular `--stack`: CLI tools conventionally use singular nouns for flags even when they accept multiple values (e.g., `docker run --volume`, `go build -tags`). The comma-separated format `--stack go,node` reads more naturally than `--stacks go,node`.

### Files to Create/Modify

- `cmd/init.go` -- Rename flags, add `--dir` and `--non-interactive`/`-y`, refactor `RunE` to use `--dir`, validate stack IDs, and add the no-stacks-detected error behavior for non-interactive mode
- `cmd/init_test.go` -- Add tests for all new flags, renamed flags, validation, edge cases

### Steps

1. **Rename `--stacks` to `--stack`** in `cmd/init.go`
   - Change `cmd.Flags().StringSliceVar(&stacks, "stacks", ...)` to `cmd.Flags().StringSliceVar(&stacks, "stack", ...)`
   - Update the help text to match

2. **Rename `--domains` to `--extra-domains`** in `cmd/init.go`
   - Change `cmd.Flags().StringSliceVar(&domains, "domains", ...)` to `cmd.Flags().StringSliceVar(&domains, "extra-domains", ...)`
   - Update the help text to clarify these are additional domains beyond the per-stack defaults

3. **Add `--dir` flag** in `cmd/init.go`
   - Add `var dir string` variable
   - Register with `cmd.Flags().StringVar(&dir, "dir", "", "Target directory (default: current directory)")`
   - In `RunE`: if `dir` is empty, fall back to `os.Getwd()`; if set, resolve to absolute path via `filepath.Abs(dir)` and validate the directory exists via `os.Stat`
   - Change `outDir` to use the resolved `dir` instead of the result of `os.Getwd()`
   - Detection also uses the resolved `dir`: `detect.Detect(dir)` already takes a path

4. **Add `--non-interactive` / `-y` flag** in `cmd/init.go`
   - Add `var nonInteractive bool` variable
   - Register with `cmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "y", false, "Skip all prompts, use detected stacks and defaults")`
   - For now, this flag is accepted but does not change behavior (there is no wizard yet). The flag establishes the API contract for when ccbox-ogj2 adds the wizard. When the wizard is added, the flow will be: if `nonInteractive` is true OR if `--stack` is provided (all required info available), skip the wizard.

5. **Add stack ID validation** in `cmd/init.go`
   - When `--stack` is provided, validate each value against `stack.IDs()` before proceeding
   - Return a clear error listing the invalid ID and the valid options: `fmt.Errorf("unknown stack %q; valid stacks: %s", s, strings.Join(validIDs, ", "))`
   - This validation currently does not exist; the unknown ID only surfaces later as an error from `render.Merge`, which has a less helpful message

6. **Improve no-stacks-detected behavior** in `cmd/init.go`
   - The current code prints to stderr and returns nil (success exit code) when no stacks are detected and `--stack` is not provided. This is wrong for scripting: a script using `ccbox init -y` expects a non-zero exit code on failure.
   - Change to return an error: `return fmt.Errorf("no stacks detected; use --stack to specify manually")`
   - Update the existing stderr message to reference `--stack` (was `--stacks`)

7. **Update existing tests** in `cmd/init_test.go`
   - `TestInitCommand_WithStacksFlag`: rename flag from `--stacks` to `--stack`

### Testing Strategy

All tests in `cmd/init_test.go` using the established pattern: `newRootCmd()` per test, `cmd.SetArgs()`, temp directories via `t.TempDir()`.

**Tests to write:**

1. **`TestInitCommand_StackFlagRenamed`** -- Verify `--stack go,node` works (replaces `TestInitCommand_WithStacksFlag` which uses the old `--stacks` name)

2. **`TestInitCommand_ExtraDomainsFlagRenamed`** -- Verify `--extra-domains api.example.com` works and the domain appears in generated firewall output

3. **`TestInitCommand_DirFlag`** -- Create a temp dir with a `go.mod`, run `ccbox init --dir <tempdir>` from a different working directory, verify `.devcontainer/` is created inside the target dir (not the working dir)

4. **`TestInitCommand_DirFlag_NonExistent`** -- Run with `--dir /nonexistent/path`, expect an error containing the path

5. **`TestInitCommand_DirFlag_NotADirectory`** -- Create a temp file, run with `--dir <tempfile>`, expect an error

6. **`TestInitCommand_DirFlag_DefaultsToWorkingDir`** -- Run without `--dir` in a temp dir with `go.mod`, verify `.devcontainer/` is created in the working dir (existing behavior preserved)

7. **`TestInitCommand_NonInteractiveFlag`** -- Verify `-y` is accepted without error. Verify `--non-interactive` is accepted without error. Both should produce the same output as without the flag (since there is no wizard yet).

8. **`TestInitCommand_InvalidStack`** -- Run with `--stack elixir`, expect an error mentioning "unknown stack" and listing valid stacks

9. **`TestInitCommand_NoStacksDetected_ReturnsError`** -- Run in an empty temp dir (no marker files), expect a non-nil error (not just stderr output)

10. **`TestInitCommand_StackAndDirCombined`** -- Run with `--stack go --dir <tempdir>`, verify generation succeeds in the target dir without needing a `go.mod` (since stacks are explicitly provided, detection is skipped)

11. **`TestInitCommand_OldFlagNames_NotAccepted`** -- Verify that `--stacks` and `--domains` are no longer recognized (Cobra returns an error for unknown flags). This prevents silent breakage if someone had scripts using the old names.

### Edge Cases

- **`--dir` with relative path**: `filepath.Abs()` resolves it relative to the process working directory. This is the expected behavior.
- **`--dir` with trailing slash**: `filepath.Abs()` normalizes this. No special handling needed.
- **`--stack` with spaces**: `StringSliceVar` handles `--stack "go, node"` by splitting on commas, but leaves spaces in the values. Need to `strings.TrimSpace` each value before validation. The current code does not do this.
- **`--stack` with empty values**: `--stack go,,node` would produce an empty string element. Should be filtered out (skip empty strings after trim).
- **`--extra-domains` with spaces**: Same trim concern. However, `firewall.Merge` already calls `strings.TrimSpace` on user extras (see `merge.go` line 77), so this is handled downstream. Still worth trimming at the flag level for consistency and clearer validation errors.
- **Duplicate stack IDs from flag**: `--stack go,go` -- `render.Merge` already deduplicates, so no special handling needed.
- **`--non-interactive` combined with `--stack`**: Both flags work independently. `--stack` skips detection regardless of `--non-interactive`. `--non-interactive` alone means "auto-detect and proceed without prompts."

### Open Questions

None -- the scope is well-defined. The `--non-interactive` flag is a no-op in the current codebase (no wizard exists), but it establishes the contract for ccbox-ogj2.

## Checklist

- [x] Tests written and passing
- [x] No TODO/FIXME/HACK/XXX comments in new code
- [x] Lint passes
- [x] Tests pass
- [x] Branch pushed and PR created (https://github.com/BjRo/ccbox/pull/13)
- [x] Automated code review passed
- [x] Review feedback worked in
- [x] User notified

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | done | 1 | 2026-04-03 |
| challenge | done (APPROVED) | 1 | 2026-04-03 |
| implement | done | 1 | 2026-04-03 |
| pr | done | 1 | 2026-04-03 |
| review | done (clean) | 2 | 2026-04-03 |
| codify | done | 1 | 2026-04-03 |
