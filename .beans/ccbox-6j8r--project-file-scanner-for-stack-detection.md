---
# ccbox-6j8r
title: Project file scanner for stack detection
status: in-progress
type: task
priority: high
created_at: 2026-04-02T10:34:14Z
updated_at: 2026-04-02T12:53:03Z
parent: ccbox-2n15
---

## Description
Implement a scanner that walks the target project directory and detects tech stacks by looking for marker files:

| Stack | Marker Files |
|-------|-------------|
| Go | `go.mod` |
| Node/TypeScript | `package.json`, `tsconfig.json` |
| Python | `requirements.txt`, `pyproject.toml`, `setup.py`, `Pipfile` |
| Rust | `Cargo.toml` |
| Ruby | `Gemfile`, `*.gemspec` |

Returns a list of detected stacks. Only scan root and one level deep (avoid vendor/node_modules).

## Checklist

- [x] Tests written (TDD)
- [x] No TODO/FIXME/HACK/XXX in code
- [x] Lint passes
- [x] Tests pass
- [x] Branch pushed
- [x] PR created
- [x] Automated code review passed
- [x] Review feedback worked in
- [ ] ADR written (if architectural)
- [ ] User notified

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | completed | 1 | 2026-04-02 | | |
| challenge | completed | 1 | 2026-04-02 | | |
| implement | completed | 1 | 2026-04-02 |
| pr | completed | 1 | 2026-04-02 |
| review | completed | 1 | 2026-04-02 |
| codify | pending | | |

## Implementation Plan

### Approach

Implement a stateless scanner function in `internal/detect/` that accepts a directory path, iterates over the stack registry, and checks for the presence of marker files at the project root and one level of subdirectories. The scanner uses `os.Stat` for exact filenames (from the registry's `MarkerFiles`) and `filepath.Glob` for pattern-based markers (like `*.gemspec` for Ruby). The detect package owns a small, internal mapping of additional glob patterns per stack, since the registry deliberately excludes patterns (per its own doc comment on `MarkerFiles`).

The function returns `[]stack.StackID` sorted alphabetically for deterministic output. An empty slice (not nil) means no stacks detected. Errors from filesystem access are returned to the caller; the scanner does not silently swallow them.

### Key Design Decisions

1. **Glob patterns live in the detect package, not the stack registry.** The `MarkerFiles` field on `stack.Stack` explicitly states it holds "exact filenames only, not glob patterns" and defers pattern-based detection to the scanner. The detect package will own a small `var globs` map that pairs `stack.StackID` with additional glob patterns (initially just `{"*.gemspec"}` for Ruby).

2. **Depth-limited scan (root + 1 level), not recursive walk.** The bean specifies "only scan root and one level deep." This avoids traversing vendor directories, node_modules, nested subprojects, etc. Implementation: check for markers at `dir/<marker>` and `dir/*/<marker>` (for exact files) or `dir/<pattern>` and `dir/*/<pattern>` (for globs). Using `filepath.Glob` handles both levels cleanly.

3. **Skip well-known noise directories.** Even at depth-1, directories like `vendor/`, `node_modules/`, `.git/`, `testdata/` could produce false positives. The scanner will maintain a small skip-set of directory names to exclude when expanding the one-level-deep glob patterns.

4. **Use `os.DirFS` + `fs.FS` interface for testability.** Accept an `fs.FS` in the core logic so tests can use `fstest.MapFS` instead of creating real temp directories. The public API function takes a `string` path and wraps it with `os.DirFS`. This gives us fast, hermetic unit tests with no filesystem side effects.

5. **Return `[]stack.StackID`, not `[]stack.Stack`.** The caller can look up full stack metadata via `stack.Get()`. Returning IDs keeps the API minimal and avoids coupling to the full Stack struct.

### Files to Create/Modify

- `internal/detect/detect.go` -- Replace stub with full scanner implementation. Contains the public `Detect(dir string) ([]stack.StackID, error)` function and the internal `detect(fsys fs.FS) ([]stack.StackID, error)` function that operates on an `fs.FS`.
- `internal/detect/detect_test.go` -- Comprehensive test suite using `testing/fstest.MapFS`.
- `internal/detect/doc.go` -- Package documentation (following the pattern from `internal/stack/doc.go`).

### Public API

```go
// Detect scans the project directory at dir and returns the IDs of all
// detected technology stacks, sorted alphabetically. It checks for marker
// files at the project root and one directory level deep, skipping well-known
// noise directories (vendor, node_modules, .git, etc.).
//
// An empty (non-nil) slice is returned when no stacks are detected.
func Detect(dir string) ([]stack.StackID, error)
```

### Internal Types and Functions

```go
// skipDirs contains directory names that should be excluded from the
// one-level-deep scan. These are well-known directories that either
// contain vendored dependencies (which would cause false positives) or
// are not part of the project source.
var skipDirs = map[string]bool{
    "vendor":       true,
    "node_modules": true,
    ".git":         true,
    "testdata":     true,
    ".devcontainer": true,
}

// globs maps stack IDs to additional glob patterns for detection.
// These patterns supplement the exact-match MarkerFiles from the stack
// registry. The registry deliberately excludes glob patterns (see
// stack.Stack.MarkerFiles doc comment), so pattern-based detection
// lives here in the scanner.
var globs = map[stack.StackID][]string{
    stack.Ruby: {"*.gemspec"},
}

// detect is the fs.FS-based core of Detect, separated for testability.
// Tests pass fstest.MapFS; the public Detect function passes os.DirFS(dir).
func detect(fsys fs.FS) ([]stack.StackID, error)

// hasMarkerFile checks whether any of the given exact filenames exist
// at the root of fsys or one level deep (excluding skipDirs).
func hasMarkerFile(fsys fs.FS, markers []string) (bool, error)

// hasGlobMatch checks whether any of the given glob patterns match
// at the root of fsys or one level deep (excluding skipDirs).
func hasGlobMatch(fsys fs.FS, patterns []string) (bool, error)

// subdirs returns the names of immediate subdirectories in fsys,
// filtering out entries in skipDirs.
func subdirs(fsys fs.FS) ([]string, error)
```

### Steps

1. **Create `internal/detect/doc.go`** -- Package-level documentation mirroring the pattern in `internal/stack/doc.go`. Brief description of the package purpose: scanning project directories for marker files to detect technology stacks.

2. **Write failing tests first (TDD red phase)** -- Create `internal/detect/detect_test.go` with all test cases using `fstest.MapFS`:

   - `TestDetect_SingleStack_Go` -- `fstest.MapFS` with `go.mod` at root. Expect `[]stack.StackID{stack.Go}`.
   - `TestDetect_SingleStack_Node` -- `package.json` at root. Expect `[]stack.StackID{stack.Node}`.
   - `TestDetect_SingleStack_Python` -- Each Python marker individually: `requirements.txt`, `pyproject.toml`, `setup.py`, `Pipfile`. Use subtests.
   - `TestDetect_SingleStack_Rust` -- `Cargo.toml` at root. Expect `[]stack.StackID{stack.Rust}`.
   - `TestDetect_SingleStack_Ruby_Gemfile` -- `Gemfile` at root. Expect `[]stack.StackID{stack.Ruby}`.
   - `TestDetect_SingleStack_Ruby_Gemspec` -- `foo.gemspec` at root (no `Gemfile`). Expect `[]stack.StackID{stack.Ruby}`. This exercises the glob pattern path.
   - `TestDetect_MultiStack` -- `go.mod` + `package.json` at root. Expect `[]stack.StackID{stack.Go, stack.Node}` (sorted).
   - `TestDetect_AllStacks` -- All five stacks present. Verify all returned, sorted.
   - `TestDetect_NoStacks` -- Empty `fstest.MapFS` or one with only unrelated files (e.g., `README.md`). Expect empty non-nil slice.
   - `TestDetect_MarkerInSubdir` -- `subproject/go.mod` one level deep. Expect `[]stack.StackID{stack.Go}`.
   - `TestDetect_MarkerTwoLevelsDeep_NotDetected` -- `deep/nested/go.mod`. Expect empty slice (too deep).
   - `TestDetect_SkipsVendorDir` -- `vendor/Cargo.toml` present. Expect empty slice (vendor is skipped).
   - `TestDetect_SkipsNodeModules` -- `node_modules/package.json` present. Expect empty slice.
   - `TestDetect_SkipsGitDir` -- `.git/go.mod` present. Expect empty slice.
   - `TestDetect_GemspecInSubdir` -- `mylib/foo.gemspec` one level deep. Expect `[]stack.StackID{stack.Ruby}`.
   - `TestDetect_GemspecInSkipDir_NotDetected` -- `vendor/foo.gemspec`. Expect empty slice.
   - `TestDetect_ResultIsSorted` -- Multiple stacks present. Verify the returned slice is sorted alphabetically.
   - `TestDetect_EmptyResult_IsNonNil` -- No stacks found. Verify `result != nil && len(result) == 0`.
   - `TestDetect_NoDuplicates` -- `go.mod` at root AND in a subdirectory. Expect Go appears only once.
   - `TestDetect_PublicAPI_InvalidDir` -- Call `Detect("/nonexistent/path")`. Expect a non-nil error.

3. **Implement the `subdirs` function** -- Read the root of `fsys` with `fs.ReadDir`, filter to directories only, exclude names in `skipDirs`, return sorted list of names.

4. **Implement `hasMarkerFile`** -- For each marker filename: first check `fs.Stat(fsys, marker)` at root. If found, return true. Then iterate over `subdirs(fsys)` and check `fs.Stat(fsys, subdir+"/"+marker)`. Return false if no match found. Use `errors.Is(err, fs.ErrNotExist)` to distinguish "not found" from real I/O errors.

5. **Implement `hasGlobMatch`** -- For each pattern: use `fs.Glob(fsys, pattern)` to check root level. Then for each subdir, use `fs.Glob(fsys, subdir+"/"+pattern)`. If any match is found, return true.

6. **Implement the core `detect` function** -- Iterate over `stack.All()`. For each stack, call `hasMarkerFile` with its `MarkerFiles`. If not found and the stack has entries in the `globs` map, call `hasGlobMatch`. If either succeeds, add the stack ID to the result. Return sorted, deduplicated `[]stack.StackID`.

7. **Implement the public `Detect` function** -- Validate the directory exists and is a directory via `os.Stat`. Create `os.DirFS(dir)` and delegate to `detect(fsys)`.

8. **Run tests, lint, verify** -- `go test ./internal/detect/...`, `golangci-lint run ./...`.

### Testing Strategy

- **Unit tests use `fstest.MapFS`** -- No real filesystem I/O needed for the core logic. This makes tests fast, hermetic, and cross-platform.
- **Table-driven subtests** where multiple inputs test the same behavior (e.g., Python markers).
- **One integration-style test** (`TestDetect_PublicAPI_InvalidDir`) exercises the public `Detect` function with a real path to verify the `os.DirFS` wiring and error handling.
- **Verify sort invariant** explicitly (not just implicitly by comparing to a pre-sorted expected value).
- **Verify non-nil empty slice** explicitly to catch accidental nil returns.

### Edge Cases

1. **`*.gemspec` glob at root vs. subdir** -- Both must work. The glob pattern `*.gemspec` matches at root; `subdir/*.gemspec` matches one level deep.
2. **Marker in both root and subdir** -- Stack should appear only once (deduplication).
3. **Marker inside a skip directory** -- Must not trigger detection (`vendor/go.mod`, `node_modules/package.json`).
4. **Marker exactly two levels deep** -- Must not trigger detection (`a/b/go.mod`).
5. **Empty directory** -- Returns empty non-nil slice, no error.
6. **Directory with no recognized markers** -- Returns empty non-nil slice, no error.
7. **Permission errors on subdirectories** -- The scanner should return the error, not silently ignore it. The caller can decide how to handle it.
8. **Symlinks** -- `fs.Stat` follows symlinks by default. `fstest.MapFS` does not support them, so this is implicitly not tested. If symlink handling becomes important, it can be addressed in a follow-up.
9. **Non-existent directory passed to `Detect`** -- Returns a clear error before attempting the scan.

### Open Questions

None. The approach is straightforward given the existing registry design. The key decision (glob patterns owned by detect, not the registry) is already documented in the `MarkerFiles` field comment in `internal/stack/stack.go` (lines 57-61).