package render

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bjro/agentbox/internal/firewall"
	"github.com/bjro/agentbox/internal/stack"
)

// GenerationConfig holds the fully merged, deduplicated configuration
// produced by combining multiple detected stacks. It is the single input
// to the template rendering pipeline.
type GenerationConfig struct {
	Stacks     []stack.StackID        // Detected stack IDs, deduplicated and sorted
	Runtimes   []stack.Runtime        // Merged runtime entries for mise.toml, sorted by Tool
	LSPs       []stack.LSP            // Merged LSP servers for Dockerfile + settings, sorted by Package
	SystemDeps []string               // Merged apt packages from all stacks, deduplicated and sorted
	Domains    firewall.MergedDomains // Merged domain allowlists (static + dynamic)
}

// Merge combines metadata from the given stacks and user-provided extra domains
// into a single GenerationConfig. It validates that all stack IDs exist in the
// registry and returns an error for the first unknown ID. Duplicate stack IDs
// in the input are deduplicated. Runtimes are deduplicated by Tool name and LSPs
// by Package name; when two stacks share a key, the alphabetically-first stack
// wins (input is sorted by StackID before collection).
func Merge(stacks []stack.StackID, userExtraDomains []string) (GenerationConfig, error) {
	// Step 1: Validate all stack IDs and deduplicate.
	seen := make(map[stack.StackID]bool)
	var uniqueStacks []stack.StackID

	for _, id := range stacks {
		if _, ok := stack.Get(id); !ok {
			return GenerationConfig{}, fmt.Errorf("render: unknown stack %q", id)
		}
		if !seen[id] {
			seen[id] = true
			uniqueStacks = append(uniqueStacks, id)
		}
	}

	// Sort the deduplicated stack IDs.
	slices.SortFunc(uniqueStacks, func(a, b stack.StackID) int {
		return strings.Compare(string(a), string(b))
	})

	// Step 2: Collect runtimes and LSPs, deduplicating by key.
	seenTools := make(map[string]bool)
	seenPackages := make(map[string]bool)
	var runtimes []stack.Runtime
	var lsps []stack.LSP

	for _, id := range uniqueStacks {
		s, _ := stack.Get(id) // Already validated above.

		if !seenTools[s.Runtime.Tool] {
			seenTools[s.Runtime.Tool] = true
			runtimes = append(runtimes, s.Runtime)
		}

		if !seenPackages[s.LSP.Package] {
			seenPackages[s.LSP.Package] = true
			lsps = append(lsps, s.LSP)
		}
	}

	// Step 3: Collect system deps, deduplicating by string value.
	seenDeps := make(map[string]bool)
	var systemDeps []string
	for _, id := range uniqueStacks {
		s, _ := stack.Get(id) // Already validated above.
		for _, dep := range s.SystemDeps {
			if !seenDeps[dep] {
				seenDeps[dep] = true
				systemDeps = append(systemDeps, dep)
			}
		}
	}
	slices.Sort(systemDeps)

	// Step 4: Sort runtimes by Tool, LSPs by Package.
	slices.SortFunc(runtimes, func(a, b stack.Runtime) int {
		return strings.Compare(a.Tool, b.Tool)
	})
	slices.SortFunc(lsps, func(a, b stack.LSP) int {
		return strings.Compare(a.Package, b.Package)
	})

	// Step 4: Delegate domain merging to firewall.Merge.
	domains, err := firewall.Merge(uniqueStacks, userExtraDomains)
	if err != nil {
		return GenerationConfig{}, fmt.Errorf("render: %w", err)
	}

	// Ensure non-nil empty slices for template-friendly zero values.
	if uniqueStacks == nil {
		uniqueStacks = []stack.StackID{}
	}
	if runtimes == nil {
		runtimes = []stack.Runtime{}
	}
	if lsps == nil {
		lsps = []stack.LSP{}
	}
	if systemDeps == nil {
		systemDeps = []string{}
	}

	return GenerationConfig{
		Stacks:     uniqueStacks,
		Runtimes:   runtimes,
		LSPs:       lsps,
		SystemDeps: systemDeps,
		Domains:    domains,
	}, nil
}
