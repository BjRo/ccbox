package detect

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/bjro/agentbox/internal/stack"
)

// skipDirs contains directory names that should be excluded from the
// one-level-deep scan. These are well-known directories that either
// contain vendored dependencies (which would cause false positives) or
// are not part of the project source.
var skipDirs = map[string]bool{
	"vendor":        true,
	"node_modules":  true,
	".git":          true,
	"testdata":      true,
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

// Detect scans the project directory at dir and returns the IDs of all
// detected technology stacks, sorted alphabetically. It checks for marker
// files at the project root and one directory level deep, skipping well-known
// noise directories (vendor, node_modules, .git, etc.).
//
// An empty (non-nil) slice is returned when no stacks are detected.
func Detect(dir string) ([]stack.StackID, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("detect: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("detect: %s is not a directory", dir)
	}

	return detect(os.DirFS(dir))
}

// detect is the fs.FS-based core of Detect, separated for testability.
// Tests pass fstest.MapFS; the public Detect function passes os.DirFS(dir).
func detect(fsys fs.FS) ([]stack.StackID, error) {
	dirs, err := subdirs(fsys)
	if err != nil {
		return nil, err
	}

	result := make([]stack.StackID, 0)

	for _, s := range stack.All() {
		found, markerErr := hasMarkerFile(fsys, s.MarkerFiles, dirs)
		if markerErr != nil {
			return nil, markerErr
		}

		if !found {
			if patterns, ok := globs[s.ID]; ok {
				found, markerErr = hasGlobMatch(fsys, patterns, dirs)
				if markerErr != nil {
					return nil, markerErr
				}
			}
		}

		if found {
			result = append(result, s.ID)
		}
	}

	// Defensive: re-sort in case stack.All() iteration order changes in
	// the future. On already-sorted input this is a no-op.
	slices.SortFunc(result, func(a, b stack.StackID) int {
		return strings.Compare(string(a), string(b))
	})

	return result, nil
}

// hasMarkerFile checks whether any of the given exact filenames exist
// at the root of fsys or one level deep (excluding skipDirs).
func hasMarkerFile(fsys fs.FS, markers []string, dirs []string) (bool, error) {
	for _, marker := range markers {
		// Check root level.
		_, err := fs.Stat(fsys, marker)
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return false, fmt.Errorf("detect: checking %s: %w", marker, err)
		}

		// Check one level deep.
		for _, dir := range dirs {
			_, err = fs.Stat(fsys, dir+"/"+marker)
			if err == nil {
				return true, nil
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return false, fmt.Errorf("detect: checking %s/%s: %w", dir, marker, err)
			}
		}
	}

	return false, nil
}

// hasGlobMatch checks whether any of the given glob patterns match
// at the root of fsys or one level deep (excluding skipDirs).
func hasGlobMatch(fsys fs.FS, patterns []string, dirs []string) (bool, error) {
	for _, pattern := range patterns {
		// Check root level.
		matches, err := fs.Glob(fsys, pattern)
		if err != nil {
			return false, fmt.Errorf("detect: glob %s: %w", pattern, err)
		}
		if len(matches) > 0 {
			return true, nil
		}

		// Check one level deep.
		for _, dir := range dirs {
			matches, err = fs.Glob(fsys, dir+"/"+pattern)
			if err != nil {
				return false, fmt.Errorf("detect: glob %s/%s: %w", dir, pattern, err)
			}
			if len(matches) > 0 {
				return true, nil
			}
		}
	}

	return false, nil
}

// subdirs returns the names of immediate subdirectories in fsys,
// filtering out entries in skipDirs.
func subdirs(fsys fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("detect: reading directory: %w", err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && !skipDirs[entry.Name()] {
			dirs = append(dirs, entry.Name())
		}
	}

	slices.Sort(dirs)
	return dirs, nil
}
