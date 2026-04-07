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
