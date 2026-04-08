package firewall

import (
	"slices"
	"testing"
)

func TestRegistry_ContainsAllStacks(t *testing.T) {
	reg := Registry()

	expectedStacks := []Stack{AlwaysOn, Go, Node, Python, Rust, Ruby}
	if len(reg) != len(expectedStacks) {
		t.Fatalf("Registry has %d stacks, want %d", len(reg), len(expectedStacks))
	}

	for _, s := range expectedStacks {
		if _, ok := reg[s]; !ok {
			t.Errorf("Registry missing stack %q", s)
		}
	}
}

func TestRegistry_AlwaysOnDomains(t *testing.T) {
	reg := Registry()
	al, ok := reg[AlwaysOn]
	if !ok {
		t.Fatal("Registry missing AlwaysOn stack")
	}

	if al.Stack != AlwaysOn {
		t.Errorf("Allowlist.Stack = %q, want %q", al.Stack, AlwaysOn)
	}

	expected := map[string]Category{
		"api.github.com":  Dynamic,
		"github.com":      Dynamic,
		"*.anthropic.com": Dynamic,
		"api.openai.com":  Dynamic,
		"auth.openai.com": Dynamic,
		"sentry.io":       Static,
		"statsig.com":     Static,
	}

	if len(al.Domains) != len(expected) {
		t.Fatalf("AlwaysOn has %d domains, want %d", len(al.Domains), len(expected))
	}

	for _, d := range al.Domains {
		wantCat, ok := expected[d.Name]
		if !ok {
			t.Errorf("unexpected domain %q in AlwaysOn", d.Name)
			continue
		}
		if d.Category != wantCat {
			t.Errorf("domain %q category = %q, want %q", d.Name, d.Category, wantCat)
		}
	}
}

func TestRegistry_PerStackDomains(t *testing.T) {
	tests := []struct {
		stack   Stack
		domains []string
	}{
		{
			stack:   Go,
			domains: []string{"proxy.golang.org", "sum.golang.org", "storage.googleapis.com"},
		},
		{
			stack:   Node,
			domains: []string{"registry.npmjs.org", "cdn.jsdelivr.net", "unpkg.com"},
		},
		{
			stack:   Python,
			domains: []string{"pypi.org", "files.pythonhosted.org"},
		},
		{
			stack:   Rust,
			domains: []string{"crates.io", "static.crates.io"},
		},
		{
			stack:   Ruby,
			domains: []string{"rubygems.org", "index.rubygems.org"},
		},
	}

	reg := Registry()

	for _, tt := range tests {
		t.Run(string(tt.stack), func(t *testing.T) {
			al, ok := reg[tt.stack]
			if !ok {
				t.Fatalf("Registry missing stack %q", tt.stack)
			}

			if al.Stack != tt.stack {
				t.Errorf("Allowlist.Stack = %q, want %q", al.Stack, tt.stack)
			}

			if len(al.Domains) != len(tt.domains) {
				t.Fatalf("stack %q has %d domains, want %d", tt.stack, len(al.Domains), len(tt.domains))
			}

			got := make(map[string]bool)
			for _, d := range al.Domains {
				got[d.Name] = true
			}

			for _, want := range tt.domains {
				if !got[want] {
					t.Errorf("stack %q missing domain %q", tt.stack, want)
				}
			}
		})
	}
}

func TestRegistry_AllDomainsHaveRationale(t *testing.T) {
	reg := Registry()

	for stack, al := range reg {
		for _, d := range al.Domains {
			if d.Rationale == "" {
				t.Errorf("stack %q, domain %q has empty Rationale", stack, d.Name)
			}
		}
	}
}

func TestRegistry_AllDomainsHaveCategory(t *testing.T) {
	reg := Registry()

	for stack, al := range reg {
		for _, d := range al.Domains {
			if d.Category != Static && d.Category != Dynamic {
				t.Errorf("stack %q, domain %q has invalid Category %q (want Static or Dynamic)", stack, d.Name, d.Category)
			}
		}
	}
}

func TestRegistry_NoDuplicateDomainsWithinStack(t *testing.T) {
	reg := Registry()

	for stack, al := range reg {
		seen := make(map[string]bool)
		for _, d := range al.Domains {
			if seen[d.Name] {
				t.Errorf("stack %q has duplicate domain %q", stack, d.Name)
			}
			seen[d.Name] = true
		}
	}
}

func TestForStack_Found(t *testing.T) {
	al, ok := ForStack(Go)
	if !ok {
		t.Fatal("ForStack(Go) returned false, want true")
	}

	if al.Stack != Go {
		t.Errorf("Allowlist.Stack = %q, want %q", al.Stack, Go)
	}

	expectedDomains := []string{"proxy.golang.org", "sum.golang.org", "storage.googleapis.com"}
	if len(al.Domains) != len(expectedDomains) {
		t.Fatalf("ForStack(Go) returned %d domains, want %d", len(al.Domains), len(expectedDomains))
	}

	got := make(map[string]bool)
	for _, d := range al.Domains {
		got[d.Name] = true
	}

	for _, want := range expectedDomains {
		if !got[want] {
			t.Errorf("ForStack(Go) missing domain %q", want)
		}
	}
}

func TestForStack_NotFound(t *testing.T) {
	_, ok := ForStack(Stack("elixir"))
	if ok {
		t.Error("ForStack(elixir) returned true, want false")
	}
}

func TestStacks_Order(t *testing.T) {
	stacks := Stacks()

	expected := []Stack{AlwaysOn, Go, Node, Python, Ruby, Rust}
	if len(stacks) != len(expected) {
		t.Fatalf("Stacks() returned %d stacks, want %d", len(stacks), len(expected))
	}

	if !slices.IsSorted(stacks) {
		t.Errorf("Stacks() not sorted: %v", stacks)
	}

	for i, want := range expected {
		if stacks[i] != want {
			t.Errorf("Stacks()[%d] = %q, want %q", i, stacks[i], want)
		}
	}
}

func TestRegistry_ReturnsDefensiveCopy(t *testing.T) {
	reg1 := Registry()
	// Mutate the returned map by deleting a key.
	delete(reg1, Go)

	reg2 := Registry()
	if _, ok := reg2[Go]; !ok {
		t.Error("mutating Registry() return value affected subsequent calls -- defensive copy is broken")
	}

	// Mutate a domain element inside the returned allowlist.
	al := reg2[AlwaysOn]
	if len(al.Domains) == 0 {
		t.Fatal("AlwaysOn has no domains")
	}
	originalName := al.Domains[0].Name
	al.Domains[0].Name = "evil.com"

	reg3 := Registry()
	if reg3[AlwaysOn].Domains[0].Name != originalName {
		t.Error("mutating Allowlist.Domains element affected canonical registry -- deep copy is broken")
	}
}

func TestForStack_ReturnsDefensiveCopy(t *testing.T) {
	al1, ok := ForStack(Go)
	if !ok {
		t.Fatal("ForStack(Go) returned false")
	}
	if len(al1.Domains) == 0 {
		t.Fatal("Go allowlist has no domains")
	}

	originalName := al1.Domains[0].Name
	al1.Domains[0].Name = "evil.com"

	al2, _ := ForStack(Go)
	if al2.Domains[0].Name != originalName {
		t.Error("mutating ForStack() return value affected canonical registry -- deep copy is broken")
	}
}
