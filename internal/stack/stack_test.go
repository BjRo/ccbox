package stack

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

func TestRegistryCompleteness(t *testing.T) {
	all := All()
	if len(all) != 5 {
		t.Fatalf("expected 5 stacks, got %d", len(all))
	}

	expectedIDs := []StackID{Go, Node, Python, Rust, Ruby}
	for _, id := range expectedIDs {
		if _, ok := Get(id); !ok {
			t.Errorf("expected stack %q to be registered", id)
		}
	}
}

func TestGet_ExistingStack(t *testing.T) {
	expectedIDs := []StackID{Go, Node, Python, Rust, Ruby}

	for _, id := range expectedIDs {
		t.Run(string(id), func(t *testing.T) {
			s, ok := Get(id)
			if !ok {
				t.Fatalf("Get(%q) returned ok=false", id)
			}
			if s.ID != id {
				t.Errorf("ID = %q, want %q", s.ID, id)
			}
			if s.Name == "" {
				t.Error("Name is empty")
			}
			if s.Runtime.Tool == "" {
				t.Error("Runtime.Tool is empty")
			}
			if s.Runtime.Version == "" {
				t.Error("Runtime.Version is empty")
			}
			if s.LSP.Package == "" {
				t.Error("LSP.Package is empty")
			}
			if s.LSP.InstallCmd == "" {
				t.Error("LSP.InstallCmd is empty")
			}
			// Plugin may be empty (e.g., Ruby has no official Claude plugin).
			if len(s.DefaultDomains) == 0 {
				t.Error("DefaultDomains is empty")
			}
			if len(s.MarkerFiles) == 0 {
				t.Error("MarkerFiles is empty")
			}
		})
	}
}

func TestGet_UnknownStack(t *testing.T) {
	_, ok := Get("unknown")
	if ok {
		t.Error("Get(\"unknown\") returned ok=true, want false")
	}
}

func TestGet_ReturnsCopies(t *testing.T) {
	first, ok := Get(Go)
	if !ok {
		t.Fatal("Get(Go) returned ok=false")
	}
	first.DefaultDomains = append(first.DefaultDomains, "evil.example.com")

	second, _ := Get(Go)
	for _, d := range second.DefaultDomains {
		if d == "evil.example.com" {
			t.Error("mutation of Get() result leaked into registry")
		}
	}
}

func TestAll_ReturnsCopies(t *testing.T) {
	first := All()
	// Mutate the first result.
	// NOTE: The Name mutation is vacuous because All() returns values, not
	// pointers, so the struct is already a shallow copy. The real guard
	// here is the DefaultDomains append below, which tests that the
	// underlying slice header was cloned (not aliased) by copyStack.
	first[0].Name = "MUTATED"
	first[0].DefaultDomains = append(first[0].DefaultDomains, "evil.example.com")

	// Second call should return unmodified data.
	second := All()
	if second[0].Name == "MUTATED" {
		t.Error("All() returned a reference to internal data; mutation was visible on second call")
	}
	for _, s := range second {
		for _, d := range s.DefaultDomains {
			if d == "evil.example.com" {
				t.Errorf("mutation of DefaultDomains leaked into registry for stack %q", s.ID)
			}
		}
	}
}

func TestAll_Sorted(t *testing.T) {
	all := All()
	ids := make([]string, len(all))
	for i, s := range all {
		ids[i] = string(s.ID)
	}
	if !slices.IsSorted(ids) {
		t.Errorf("All() is not sorted by ID: got %v", ids)
	}
}

func TestIDs_Sorted(t *testing.T) {
	ids := IDs()
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = string(id)
	}
	if !slices.IsSorted(strs) {
		t.Errorf("IDs() is not sorted: got %v", strs)
	}
}

func TestIDs_MatchesAll(t *testing.T) {
	ids := IDs()
	all := All()

	if len(ids) != len(all) {
		t.Fatalf("IDs() returned %d items, All() returned %d", len(ids), len(all))
	}

	allIDs := make(map[StackID]bool)
	for _, s := range all {
		allIDs[s.ID] = true
	}
	for _, id := range ids {
		if !allIDs[id] {
			t.Errorf("IDs() contains %q which is not in All()", id)
		}
	}
}

func TestNoDuplicateMarkerFiles(t *testing.T) {
	all := All()
	seen := make(map[string]StackID)
	for _, s := range all {
		for _, mf := range s.MarkerFiles {
			if other, exists := seen[mf]; exists {
				t.Errorf("marker file %q is claimed by both %q and %q", mf, other, s.ID)
			}
			seen[mf] = s.ID
		}
	}
}

func TestNoDuplicateDomains(t *testing.T) {
	all := All()
	for _, s := range all {
		t.Run(string(s.ID)+"/DefaultDomains", func(t *testing.T) {
			seen := make(map[string]bool)
			for _, d := range s.DefaultDomains {
				if seen[d] {
					t.Errorf("duplicate default domain %q in stack %q", d, s.ID)
				}
				seen[d] = true
			}
		})
		t.Run(string(s.ID)+"/DynamicDomains", func(t *testing.T) {
			seen := make(map[string]bool)
			for _, d := range s.DynamicDomains {
				if seen[d] {
					t.Errorf("duplicate dynamic domain %q in stack %q", d, s.ID)
				}
				seen[d] = true
			}
		})
	}
}

func TestSystemDeps_NonNil(t *testing.T) {
	all := All()
	for _, s := range all {
		t.Run(string(s.ID), func(t *testing.T) {
			if s.SystemDeps == nil {
				t.Errorf("SystemDeps is nil for stack %q, want non-nil (possibly empty) slice", s.ID)
			}
		})
	}
}

func TestSystemDeps_DefensiveCopy(t *testing.T) {
	// Get a stack that has system deps (Ruby has several).
	first, ok := Get(Ruby)
	if !ok {
		t.Fatal("Get(Ruby) returned ok=false")
	}
	if len(first.SystemDeps) == 0 {
		t.Fatal("Ruby should have non-empty SystemDeps")
	}

	// Mutate the returned slice.
	first.SystemDeps = append(first.SystemDeps, "evil-package")

	// A second Get should not see the mutation.
	second, _ := Get(Ruby)
	for _, dep := range second.SystemDeps {
		if dep == "evil-package" {
			t.Error("mutation of SystemDeps leaked into registry")
		}
	}
}

func TestSystemDeps_KnownValues(t *testing.T) {
	// Spot-check: Ruby should include libssl-dev and libreadline-dev.
	ruby, ok := Get(Ruby)
	if !ok {
		t.Fatal("Get(Ruby) returned ok=false")
	}
	if !slices.Contains(ruby.SystemDeps, "libssl-dev") {
		t.Error("Ruby SystemDeps should contain libssl-dev")
	}
	if !slices.Contains(ruby.SystemDeps, "libreadline-dev") {
		t.Error("Ruby SystemDeps should contain libreadline-dev")
	}

	// Spot-check: Go should have empty SystemDeps.
	goStack, ok := Get(Go)
	if !ok {
		t.Fatal("Get(Go) returned ok=false")
	}
	if len(goStack.SystemDeps) != 0 {
		t.Errorf("Go SystemDeps should be empty, got %v", goStack.SystemDeps)
	}
}

func TestDevTools_NonNil(t *testing.T) {
	all := All()
	for _, s := range all {
		t.Run(string(s.ID), func(t *testing.T) {
			if s.DevTools == nil {
				t.Errorf("DevTools is nil for stack %q, want non-nil (possibly empty) slice", s.ID)
			}
		})
	}
}

func TestDevTools_DefensiveCopy(t *testing.T) {
	first, ok := Get(Go)
	if !ok {
		t.Fatal("Get(Go) returned ok=false")
	}
	if len(first.DevTools) == 0 {
		t.Fatal("Go should have non-empty DevTools")
	}

	first.DevTools = append(first.DevTools, "evil-tool install")

	second, _ := Get(Go)
	for _, dt := range second.DevTools {
		if dt == "evil-tool install" {
			t.Error("mutation of DevTools leaked into registry")
		}
	}
}

func TestDevTools_KnownValues(t *testing.T) {
	goStack, ok := Get(Go)
	if !ok {
		t.Fatal("Get(Go) returned ok=false")
	}
	found := false
	for _, dt := range goStack.DevTools {
		if strings.Contains(dt, "golangci-lint/v2") {
			found = true
		}
	}
	if !found {
		t.Error("Go DevTools should contain a golangci-lint install command")
	}

	nodeStack, ok := Get(Node)
	if !ok {
		t.Fatal("Get(Node) returned ok=false")
	}
	if len(nodeStack.DevTools) != 0 {
		t.Errorf("Node DevTools should be empty, got %v", nodeStack.DevTools)
	}
}

func TestDevTools_NoDuplicates(t *testing.T) {
	all := All()
	for _, s := range all {
		t.Run(string(s.ID), func(t *testing.T) {
			seen := make(map[string]bool)
			for _, dt := range s.DevTools {
				if seen[dt] {
					t.Errorf("duplicate dev tool %q in stack %q", dt, s.ID)
				}
				seen[dt] = true
			}
		})
	}
}

func TestDomainsAreValidHostnames(t *testing.T) {
	// A basic hostname pattern: one or more labels separated by dots.
	// Each label is alphanumeric with optional hyphens, no leading/trailing hyphens.
	hostnameRe := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)

	all := All()
	for _, s := range all {
		for _, d := range s.DefaultDomains {
			if !hostnameRe.MatchString(d) {
				t.Errorf("stack %q: default domain %q is not a valid hostname", s.ID, d)
			}
		}
		for _, d := range s.DynamicDomains {
			if !hostnameRe.MatchString(d) {
				t.Errorf("stack %q: dynamic domain %q is not a valid hostname", s.ID, d)
			}
		}
	}
}
