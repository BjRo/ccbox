package stack

import (
	"regexp"
	"sort"
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
			if s.LSP.Plugin == "" {
				t.Error("LSP.Plugin is empty")
			}
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

func TestAll_ReturnsCopies(t *testing.T) {
	first := All()
	// Mutate the first result.
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
	if !sort.StringsAreSorted(ids) {
		t.Errorf("All() is not sorted by ID: got %v", ids)
	}
}

func TestIDs_Sorted(t *testing.T) {
	ids := IDs()
	strs := make([]string, len(ids))
	for i, id := range ids {
		strs[i] = string(id)
	}
	if !sort.StringsAreSorted(strs) {
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
