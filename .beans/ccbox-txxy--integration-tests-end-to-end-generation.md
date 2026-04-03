---
# ccbox-txxy
title: 'Integration tests: end-to-end generation'
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:36:03Z
updated_at: 2026-04-03T08:38:29Z
parent: ccbox-6g75
---

## Description
End-to-end tests that run `ccbox init` against sample project directories and validate the full output:

1. Create temp dir with go.mod → run ccbox init --non-interactive → verify all 8 files exist with correct content
2. Create temp dir with package.json + go.mod → run ccbox init --non-interactive → verify multi-stack output
3. Create temp dir with existing .devcontainer/ → run ccbox init → verify it aborts with error message
4. Create temp dir with go.mod → run ccbox init --extra-domains "api.example.com" → verify domain appears in dynamic-domains.conf
5. Verify .ccbox.yml is written with correct stacks and options

Run as `go test -tags=integration` to separate from unit tests.

## Implementation Plan

### Approach

Create a new test file `cmd/init_integration_test.go` guarded by `//go:build integration` that exercises `ccbox init` through the Cobra command tree. Each test calls `newRootCmd(nil)` with appropriate `SetArgs`, pointing `--dir` at a `t.TempDir()` containing stack marker files. Tests validate the full generated output: file existence, file permissions, content correctness (via structural assertions), and config file round-trip.

Scenario 3 (existing `.devcontainer/` guard) requires a small production code change in `cmd/init.go` -- adding a pre-existence check before `os.MkdirAll`. This guard is a natural prerequisite for the integration test and is trivial enough to include in this bean.

All tests are `t.Parallel()` safe because each uses its own `t.TempDir()` and `--dir` flag (no `os.Chdir`).

### Files to Create/Modify

- `cmd/init_integration_test.go` (NEW) -- All 5 integration test functions, guarded with `//go:build integration`.
- `cmd/init.go` (MODIFY) -- Add `.devcontainer/` pre-existence guard (6-8 lines) before the `os.MkdirAll` call.

### Steps

#### 1. Add `.devcontainer/` pre-existence guard to `cmd/init.go`

Insert a check between the render calls and the `os.MkdirAll` call (around line 129). Before creating the output directory:

```go
outDir := filepath.Join(targetDir, ".devcontainer")
if info, err := os.Stat(outDir); err == nil && info.IsDir() {
    return fmt.Errorf(".devcontainer/ already exists in %s; remove it first or use a different directory", targetDir)
}
```

This uses the same error pattern as other guard checks in the codebase (e.g., `resolveDir`). The error message is actionable -- it tells the user what to do.

#### 2. Create `cmd/init_integration_test.go` with build tag

File header:
```go
//go:build integration

package cmd
```

The test file lives in `package cmd` (same as existing tests) to access the unexported `newRootCmd` constructor. This follows the established pattern from `cmd/init_test.go`.

#### 3. Test: Single Go stack end-to-end (`TestIntegration_SingleGoStack`)

Setup:
- `t.TempDir()` with a `go.mod` file (`module example\n`).
- `newRootCmd(nil)` with `SetArgs([]string{"init", "--dir", dir, "--non-interactive"})`.

Assertions:
- **File existence**: All 8 files exist in `.devcontainer/`: `Dockerfile`, `devcontainer.json`, `init-firewall.sh`, `warmup-dns.sh`, `dynamic-domains.conf`, `claude-user-settings.json`, `sync-claude-settings.sh`, `README.md`.
- **File non-empty**: `os.Stat` each file, verify `Size() > 0`.
- **Dockerfile content**: Contains `go = "latest"` (Go runtime in mise config). Contains `go install golang.org/x/tools/gopls@latest` (LSP install). Does NOT contain `python`, `ruby`, or `rust` runtime entries.
- **devcontainer.json**: Valid JSON (`json.Unmarshal` succeeds). Contains `"dockerfile": "Dockerfile"`.
- **init-firewall.sh**: Contains `proxy.golang.org` (Go static domain). Is executable (`0755` mode bits check via `os.Stat` mode).
- **dynamic-domains.conf**: Contains `proxy.golang.org` (Go dynamic domain from firewall registry).
- **README.md**: Contains `- go` (stack listed in detected stacks section).
- **Shell script permissions**: `init-firewall.sh`, `warmup-dns.sh`, `sync-claude-settings.sh` all have executable bit set (check `info.Mode().Perm() & 0o111 != 0`).

#### 4. Test: Multi-stack Go + Node (`TestIntegration_MultiStack`)

Setup:
- `t.TempDir()` with both `go.mod` and `package.json` files.
- `newRootCmd(nil)` with `SetArgs([]string{"init", "--dir", dir, "--non-interactive"})`.

Assertions:
- **All 8 files exist** (same check as scenario 1).
- **Dockerfile content**: Contains both `go = "latest"` and does NOT contain a separate `node = "lts"` in the range block (Node is hardcoded, not in the range). Contains both `go install golang.org/x/tools/gopls@latest` and `npm install -g typescript-language-server typescript`.
- **init-firewall.sh**: Contains Go domains (`proxy.golang.org`). Contains Node domain (`registry.npmjs.org`).
- **dynamic-domains.conf**: Contains both Go and Node dynamic domains.
- **README.md**: Contains both `- go` and `- node`.
- **claude-user-settings.json**: Valid JSON. `enabledPlugins` array contains both `"gopls"` and `"typescript"`.

#### 5. Test: Existing `.devcontainer/` aborts (`TestIntegration_ExistingDevcontainerAborts`)

Setup:
- `t.TempDir()` with `go.mod` AND a pre-created `.devcontainer/` directory (`os.MkdirAll`).
- `newRootCmd(nil)` with `SetArgs([]string{"init", "--dir", dir, "--non-interactive"})`.

Assertions:
- `cmd.Execute()` returns a non-nil error.
- Error message contains `".devcontainer/ already exists"`.
- No files were written inside `.devcontainer/` (directory remains empty -- `os.ReadDir` returns 0 entries).

#### 6. Test: Extra domains (`TestIntegration_ExtraDomains`)

Setup:
- `t.TempDir()` with `go.mod`.
- `newRootCmd(nil)` with `SetArgs([]string{"init", "--dir", dir, "--non-interactive", "--extra-domains", "api.example.com"})`.

Assertions:
- **All 8 files exist**.
- **dynamic-domains.conf**: Contains `api.example.com` (user extra domains are classified as Dynamic by `firewall.Merge`).
- **README.md**: Contains `api.example.com` in the dynamic domains table.

#### 7. Test: Config file correctness (`TestIntegration_ConfigFile`)

Setup:
- `t.TempDir()` with `go.mod`.
- `newRootCmd(nil)` with `SetArgs([]string{"init", "--dir", dir, "--non-interactive", "--extra-domains", "api.example.com"})`.

Assertions:
- `.ccbox.yml` exists at `filepath.Join(dir, ".ccbox.yml")`.
- Round-trip via `config.Load`: read the file, `config.Load` succeeds.
- `cfg.Version == 1`.
- `cfg.Stacks` contains `"go"` and has length 1.
- `cfg.ExtraDomains` contains `"api.example.com"` and has length 1.
- `cfg.CcboxVersion == "dev"` (the default in test builds).
- `cfg.GeneratedAt` is within the last 5 seconds of `time.Now()`.

### Testing Strategy

- **Build tag isolation**: `//go:build integration` ensures `go test ./...` skips these tests by default. Run with `go test -tags=integration ./cmd/...` explicitly.
- **Parallel execution**: Every test uses `t.Parallel()` + `--dir` flag + `t.TempDir()`. No shared mutable state, no `os.Chdir`.
- **Structural assertions**: No golden-file comparisons. Each test checks structural properties (file existence, content substrings, JSON validity, config round-trip) that are resilient to template wording changes.
- **Registry-grounded checks**: Domain assertions are checked against known registry entries (e.g., `proxy.golang.org` for Go) rather than hardcoded counts.
- **Permission checks**: Shell scripts verified for executable bits using `info.Mode().Perm()`.
- **No test doubles**: Integration tests use `newRootCmd(nil)` with no fake prompter. The `--non-interactive` flag (or non-TTY stdin) ensures the wizard is skipped.

### Helpers

Define a shared `assertFileExists(t *testing.T, path string) os.FileInfo` helper at the top of the integration test file to reduce boilerplate across tests. This helper calls `t.Helper()`, stats the file, and fails with `t.Fatalf` if missing.

Define a `readFile(t *testing.T, path string) string` helper that reads a file and fatals on error.

### Edge Cases

- **Go runtime in Dockerfile**: The mise config always includes `node = "lts"` hardcoded. When Go is detected, the template range only adds `go = "latest"`. Tests must not falsely expect `node` in the range output.
- **Domain category**: `proxy.golang.org` is `Dynamic` in the firewall registry (despite being `DefaultDomains` in the stack registry -- these are different data). The integration test checks it appears in `dynamic-domains.conf`, not in the static section of `init-firewall.sh`.
- **Extra domains are always Dynamic**: Per `firewall.Merge`, user extras are classified as Dynamic. The test for `api.example.com` checks `dynamic-domains.conf`, not `init-firewall.sh` static resolution.

### Open Questions

None. The implementation plan is fully grounded in the current codebase.

## Checklist

- [ ] Tests written (TDD)
- [ ] No TODO/FIXME/HACK/XXX comments
- [ ] Lint passes
- [ ] Tests pass
- [ ] Branch pushed
- [ ] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] All other checklist items completed
- [ ] User notified

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | complete | 1 | 2026-04-03 |
| challenge | pending | | |
| implement | pending | | |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |
