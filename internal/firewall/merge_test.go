package firewall

import (
	"slices"
	"strings"
	"testing"

	"github.com/bjro/ccbox/internal/stack"
)

// collectExpected gathers all unique domains from ForStack for the given
// firewall stacks, returning maps of expected static and dynamic domain names.
// This allows tests to verify structural invariants computed from the registry
// rather than hardcoded counts.
func collectExpected(stacks ...Stack) (staticNames, dynamicNames map[string]bool) {
	staticNames = make(map[string]bool)
	dynamicNames = make(map[string]bool)
	seen := make(map[string]bool)

	for _, s := range stacks {
		al, ok := ForStack(s)
		if !ok {
			continue
		}
		for _, d := range al.Domains {
			if seen[d.Name] {
				continue
			}
			seen[d.Name] = true
			switch d.Category {
			case Static:
				staticNames[d.Name] = true
			case Dynamic:
				dynamicNames[d.Name] = true
			}
		}
	}
	return staticNames, dynamicNames
}

func TestMerge_AlwaysOnIncluded(t *testing.T) {
	result := Merge(nil, nil)

	wantStatic, wantDynamic := collectExpected(AlwaysOn)

	if len(result.Static) != len(wantStatic) {
		t.Fatalf("Static count = %d, want %d", len(result.Static), len(wantStatic))
	}
	if len(result.Dynamic) != len(wantDynamic) {
		t.Fatalf("Dynamic count = %d, want %d", len(result.Dynamic), len(wantDynamic))
	}

	for _, d := range result.Static {
		if !wantStatic[d.Name] {
			t.Errorf("unexpected static domain %q", d.Name)
		}
	}
	for _, d := range result.Dynamic {
		if !wantDynamic[d.Name] {
			t.Errorf("unexpected dynamic domain %q", d.Name)
		}
	}

	// Verify sorted order within each list.
	if !slices.IsSortedFunc(result.Static, func(a, b Domain) int {
		return strings.Compare(a.Name, b.Name)
	}) {
		t.Error("Static domains not sorted")
	}
	if !slices.IsSortedFunc(result.Dynamic, func(a, b Domain) int {
		return strings.Compare(a.Name, b.Name)
	}) {
		t.Error("Dynamic domains not sorted")
	}
}

func TestMerge_SingleStack(t *testing.T) {
	result := Merge([]stack.StackID{stack.Go}, nil)

	// Collect expected domains from AlwaysOn + Go.
	wantStatic, wantDynamic := collectExpected(AlwaysOn, Go)

	if len(result.Static) != len(wantStatic) {
		t.Fatalf("Static count = %d, want %d", len(result.Static), len(wantStatic))
	}
	if len(result.Dynamic) != len(wantDynamic) {
		t.Fatalf("Dynamic count = %d, want %d", len(result.Dynamic), len(wantDynamic))
	}

	allDomains := make(map[string]bool)
	for _, d := range result.Static {
		allDomains[d.Name] = true
	}
	for _, d := range result.Dynamic {
		allDomains[d.Name] = true
	}

	// Verify Go-specific domains are present.
	goAl, _ := ForStack(Go)
	for _, d := range goAl.Domains {
		if !allDomains[d.Name] {
			t.Errorf("missing Go domain %q", d.Name)
		}
	}

	// Verify no Node domains are present (unless they overlap with AlwaysOn or Go).
	nodeAl, _ := ForStack(Node)
	for _, d := range nodeAl.Domains {
		if allDomains[d.Name] && !wantStatic[d.Name] && !wantDynamic[d.Name] {
			t.Errorf("unexpected Node domain %q in single-Go merge", d.Name)
		}
	}
}

func TestMerge_MultipleStacks(t *testing.T) {
	result := Merge([]stack.StackID{stack.Go, stack.Node}, nil)

	wantStatic, wantDynamic := collectExpected(AlwaysOn, Go, Node)

	if len(result.Static) != len(wantStatic) {
		t.Fatalf("Static count = %d, want %d", len(result.Static), len(wantStatic))
	}
	if len(result.Dynamic) != len(wantDynamic) {
		t.Fatalf("Dynamic count = %d, want %d", len(result.Dynamic), len(wantDynamic))
	}

	// Verify no duplicates across Static and Dynamic.
	seen := make(map[string]bool)
	for _, d := range result.Static {
		if seen[d.Name] {
			t.Errorf("duplicate domain %q in Static", d.Name)
		}
		seen[d.Name] = true
	}
	for _, d := range result.Dynamic {
		if seen[d.Name] {
			t.Errorf("duplicate domain %q across Static/Dynamic", d.Name)
		}
		seen[d.Name] = true
	}
}

func TestMerge_UserExtras(t *testing.T) {
	result := Merge(nil, []string{"custom.example.com", "another.example.com"})

	// User extras should appear in Dynamic list.
	dynamicNames := make(map[string]bool)
	for _, d := range result.Dynamic {
		dynamicNames[d.Name] = true
	}

	if !dynamicNames["custom.example.com"] {
		t.Error("missing user extra custom.example.com in Dynamic")
	}
	if !dynamicNames["another.example.com"] {
		t.Error("missing user extra another.example.com in Dynamic")
	}

	// Verify user extras have correct category and rationale.
	for _, d := range result.Dynamic {
		if d.Name == "custom.example.com" || d.Name == "another.example.com" {
			if d.Category != Dynamic {
				t.Errorf("user extra %q has category %q, want Dynamic", d.Name, d.Category)
			}
			if d.Rationale != "User-specified domain" {
				t.Errorf("user extra %q has rationale %q, want %q", d.Name, d.Rationale, "User-specified domain")
			}
		}
	}

	// Verify sorted order.
	if !slices.IsSortedFunc(result.Dynamic, func(a, b Domain) int {
		return strings.Compare(a.Name, b.Name)
	}) {
		t.Error("Dynamic domains not sorted")
	}
}

func TestMerge_UserExtraDuplicatesRegistry(t *testing.T) {
	// proxy.golang.org is in the Go registry.
	result := Merge([]stack.StackID{stack.Go}, []string{"proxy.golang.org"})

	count := 0
	for _, d := range result.Dynamic {
		if d.Name == "proxy.golang.org" {
			count++
			// Should retain the registry rationale, not the user-specified one.
			if d.Rationale == "User-specified domain" {
				t.Error("proxy.golang.org has user rationale; registry rationale should win")
			}
		}
	}
	for _, d := range result.Static {
		if d.Name == "proxy.golang.org" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("proxy.golang.org appears %d times, want 1", count)
	}
}

func TestMerge_UserExtraDuplicatesAlwaysOn(t *testing.T) {
	result := Merge(nil, []string{"github.com"})

	// github.com is Static in AlwaysOn; it should stay in Static, not move to Dynamic.
	staticCount := 0
	dynamicCount := 0
	for _, d := range result.Static {
		if d.Name == "github.com" {
			staticCount++
		}
	}
	for _, d := range result.Dynamic {
		if d.Name == "github.com" {
			dynamicCount++
		}
	}

	if staticCount != 1 {
		t.Errorf("github.com in Static %d times, want 1", staticCount)
	}
	if dynamicCount != 0 {
		t.Errorf("github.com in Dynamic %d times, want 0", dynamicCount)
	}
}

func TestMerge_DeduplicateUserExtras(t *testing.T) {
	result := Merge(nil, []string{"dup.example.com", "dup.example.com"})

	count := 0
	for _, d := range result.Dynamic {
		if d.Name == "dup.example.com" {
			count++
		}
	}

	if count != 1 {
		t.Errorf("dup.example.com appears %d times, want 1", count)
	}
}

func TestMerge_UnknownStackSkipped(t *testing.T) {
	result := Merge([]stack.StackID{"elixir"}, nil)

	// Only always-on domains should appear.
	wantStatic, wantDynamic := collectExpected(AlwaysOn)

	if len(result.Static) != len(wantStatic) {
		t.Fatalf("Static count = %d, want %d (only AlwaysOn expected)", len(result.Static), len(wantStatic))
	}
	if len(result.Dynamic) != len(wantDynamic) {
		t.Fatalf("Dynamic count = %d, want %d (only AlwaysOn expected)", len(result.Dynamic), len(wantDynamic))
	}
}

func TestMerge_EmptyUserExtrasSkipped(t *testing.T) {
	result := Merge(nil, []string{"", "  ", "valid.example.com"})

	// Only valid.example.com should appear from user extras.
	userDomains := make(map[string]bool)
	alwaysOnNames := make(map[string]bool)
	al, _ := ForStack(AlwaysOn)
	for _, d := range al.Domains {
		alwaysOnNames[d.Name] = true
	}

	for _, d := range result.Dynamic {
		if !alwaysOnNames[d.Name] {
			userDomains[d.Name] = true
		}
	}

	if len(userDomains) != 1 {
		t.Fatalf("expected 1 user domain in Dynamic, got %d: %v", len(userDomains), userDomains)
	}
	if !userDomains["valid.example.com"] {
		t.Error("missing valid.example.com in Dynamic")
	}
}

func TestMerge_SortedOutput(t *testing.T) {
	result := Merge([]stack.StackID{stack.Go, stack.Node, stack.Python}, nil)

	if !slices.IsSortedFunc(result.Static, func(a, b Domain) int {
		return strings.Compare(a.Name, b.Name)
	}) {
		names := make([]string, len(result.Static))
		for i, d := range result.Static {
			names[i] = d.Name
		}
		t.Errorf("Static domains not sorted: %v", names)
	}

	if !slices.IsSortedFunc(result.Dynamic, func(a, b Domain) int {
		return strings.Compare(a.Name, b.Name)
	}) {
		names := make([]string, len(result.Dynamic))
		for i, d := range result.Dynamic {
			names[i] = d.Name
		}
		t.Errorf("Dynamic domains not sorted: %v", names)
	}
}

func TestMerge_DuplicateStackIDs(t *testing.T) {
	single := Merge([]stack.StackID{stack.Go}, nil)
	double := Merge([]stack.StackID{stack.Go, stack.Go}, nil)

	if len(single.Static) != len(double.Static) {
		t.Errorf("Static: single=%d, double=%d -- duplicate stack ID caused different count",
			len(single.Static), len(double.Static))
	}
	if len(single.Dynamic) != len(double.Dynamic) {
		t.Errorf("Dynamic: single=%d, double=%d -- duplicate stack ID caused different count",
			len(single.Dynamic), len(double.Dynamic))
	}
}

func TestMerge_AllStacks(t *testing.T) {
	allIDs := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}
	result := Merge(allIDs, nil)

	// Compute expected unique domains from all stacks + AlwaysOn.
	firewallStacks := []Stack{AlwaysOn, Go, Node, Python, Rust, Ruby}
	wantStatic, wantDynamic := collectExpected(firewallStacks...)

	if len(result.Static) != len(wantStatic) {
		t.Errorf("Static count = %d, want %d", len(result.Static), len(wantStatic))
	}
	if len(result.Dynamic) != len(wantDynamic) {
		t.Errorf("Dynamic count = %d, want %d", len(result.Dynamic), len(wantDynamic))
	}

	// Verify total unique count matches.
	totalResult := len(result.Static) + len(result.Dynamic)
	totalExpected := len(wantStatic) + len(wantDynamic)
	if totalResult != totalExpected {
		t.Errorf("total unique domains = %d, want %d", totalResult, totalExpected)
	}

	// Verify no duplicates across the result.
	seen := make(map[string]bool)
	for _, d := range result.Static {
		if seen[d.Name] {
			t.Errorf("duplicate domain %q in result", d.Name)
		}
		seen[d.Name] = true
	}
	for _, d := range result.Dynamic {
		if seen[d.Name] {
			t.Errorf("duplicate domain %q across Static/Dynamic", d.Name)
		}
		seen[d.Name] = true
	}
}

func TestMerge_UserExtraWhitespaceTrimmed(t *testing.T) {
	result := Merge(nil, []string{"  trimmed.example.com  "})

	found := false
	for _, d := range result.Dynamic {
		if d.Name == "trimmed.example.com" {
			found = true
		}
		if d.Name == "  trimmed.example.com  " {
			t.Error("domain name was not trimmed")
		}
	}

	if !found {
		t.Error("trimmed.example.com not found in Dynamic")
	}
}
