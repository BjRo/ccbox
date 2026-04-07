package render

import (
	"slices"
	"strings"
	"testing"

	"github.com/bjro/agentbox/internal/stack"
)

func TestEnsureNode_InjectsWhenMissing(t *testing.T) {
	t.Parallel()
	cfg := &GenerationConfig{
		Runtimes: []stack.Runtime{{Tool: "go", Version: "latest"}},
	}
	EnsureNode(cfg)

	if len(cfg.Runtimes) != 2 {
		t.Fatalf("Runtimes length = %d, want 2", len(cfg.Runtimes))
	}

	var found bool
	for _, r := range cfg.Runtimes {
		if r.Tool == "node" {
			found = true
			if r.Version != "lts" {
				t.Errorf("node version = %q, want %q", r.Version, "lts")
			}
		}
	}
	if !found {
		t.Error("node runtime not found after EnsureNode")
	}
}

func TestEnsureNode_NoOpWhenPresent(t *testing.T) {
	t.Parallel()
	cfg := &GenerationConfig{
		Runtimes: []stack.Runtime{{Tool: "node", Version: "20"}},
	}
	EnsureNode(cfg)

	if len(cfg.Runtimes) != 1 {
		t.Fatalf("Runtimes length = %d, want 1", len(cfg.Runtimes))
	}
	if cfg.Runtimes[0].Version != "20" {
		t.Errorf("node version = %q, want %q", cfg.Runtimes[0].Version, "20")
	}
}

func TestEnsureNode_PreservesExistingNodeVersion(t *testing.T) {
	t.Parallel()
	cfg := &GenerationConfig{
		Runtimes: []stack.Runtime{
			{Tool: "go", Version: "latest"},
			{Tool: "node", Version: "18"},
		},
	}
	EnsureNode(cfg)

	for _, r := range cfg.Runtimes {
		if r.Tool == "node" {
			if r.Version != "18" {
				t.Errorf("node version = %q, want %q", r.Version, "18")
			}
			return
		}
	}
	t.Error("node runtime not found after EnsureNode")
}

func TestEnsureNode_MaintainsSortOrder(t *testing.T) {
	t.Parallel()
	cfg := &GenerationConfig{
		Runtimes: []stack.Runtime{
			{Tool: "ruby", Version: "latest"},
			{Tool: "go", Version: "latest"},
		},
	}
	EnsureNode(cfg)

	if len(cfg.Runtimes) != 3 {
		t.Fatalf("Runtimes length = %d, want 3", len(cfg.Runtimes))
	}

	isSorted := slices.IsSortedFunc(cfg.Runtimes, func(a, b stack.Runtime) int {
		return strings.Compare(a.Tool, b.Tool)
	})
	if !isSorted {
		tools := make([]string, len(cfg.Runtimes))
		for i, r := range cfg.Runtimes {
			tools[i] = r.Tool
		}
		t.Errorf("Runtimes not sorted by Tool: %v", tools)
	}

	// Verify order is go, node, ruby.
	expected := []string{"go", "node", "ruby"}
	for i, r := range cfg.Runtimes {
		if r.Tool != expected[i] {
			t.Errorf("Runtimes[%d].Tool = %q, want %q", i, r.Tool, expected[i])
		}
	}
}

func TestEnsureNode_EmptyRuntimes(t *testing.T) {
	t.Parallel()
	cfg := &GenerationConfig{
		Runtimes: []stack.Runtime{},
	}
	EnsureNode(cfg)

	if len(cfg.Runtimes) != 1 {
		t.Fatalf("Runtimes length = %d, want 1", len(cfg.Runtimes))
	}
	if cfg.Runtimes[0].Tool != "node" || cfg.Runtimes[0].Version != "lts" {
		t.Errorf("runtime = %+v, want {Tool: node, Version: lts}", cfg.Runtimes[0])
	}
}

func TestEnsureNode_NilRuntimes(t *testing.T) {
	t.Parallel()
	cfg := &GenerationConfig{
		Runtimes: nil,
	}
	EnsureNode(cfg)

	if cfg.Runtimes == nil {
		t.Fatal("Runtimes should not be nil after EnsureNode")
	}
	if len(cfg.Runtimes) != 1 {
		t.Fatalf("Runtimes length = %d, want 1", len(cfg.Runtimes))
	}
	if cfg.Runtimes[0].Tool != "node" || cfg.Runtimes[0].Version != "lts" {
		t.Errorf("runtime = %+v, want {Tool: node, Version: lts}", cfg.Runtimes[0])
	}
}

func TestEnsureNode_Idempotent(t *testing.T) {
	t.Parallel()
	cfg := &GenerationConfig{
		Runtimes: []stack.Runtime{{Tool: "go", Version: "latest"}},
	}

	EnsureNode(cfg)
	first := slices.Clone(cfg.Runtimes)

	EnsureNode(cfg)

	if len(cfg.Runtimes) != len(first) {
		t.Fatalf("Runtimes length changed: first=%d, second=%d", len(first), len(cfg.Runtimes))
	}
	for i := range cfg.Runtimes {
		if cfg.Runtimes[i] != first[i] {
			t.Errorf("Runtimes[%d] changed: first=%+v, second=%+v", i, first[i], cfg.Runtimes[i])
		}
	}
}
