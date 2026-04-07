package render

import (
	"slices"
	"strings"
	"testing"

	"github.com/bjro/agentbox/internal/firewall"
	"github.com/bjro/agentbox/internal/stack"
)

func TestMerge_SingleStack(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Runtimes) != 1 {
		t.Fatalf("Runtimes count = %d, want 1", len(cfg.Runtimes))
	}
	if cfg.Runtimes[0].Tool != "go" {
		t.Errorf("Runtimes[0].Tool = %q, want %q", cfg.Runtimes[0].Tool, "go")
	}

	if len(cfg.LSPs) != 1 {
		t.Fatalf("LSPs count = %d, want 1", len(cfg.LSPs))
	}
	if cfg.LSPs[0].Package != "gopls" {
		t.Errorf("LSPs[0].Package = %q, want %q", cfg.LSPs[0].Package, "gopls")
	}

	// Domains should match firewall.Merge for the same inputs.
	wantDomains, fwErr := firewall.Merge([]stack.StackID{stack.Go}, nil)
	if fwErr != nil {
		t.Fatalf("firewall.Merge unexpected error: %v", fwErr)
	}
	if len(cfg.Domains.Static) != len(wantDomains.Static) {
		t.Errorf("Static domains count = %d, want %d", len(cfg.Domains.Static), len(wantDomains.Static))
	}
	if len(cfg.Domains.Dynamic) != len(wantDomains.Dynamic) {
		t.Errorf("Dynamic domains count = %d, want %d", len(cfg.Domains.Dynamic), len(wantDomains.Dynamic))
	}
}

func TestMerge_MultipleStacks(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Runtimes) != 2 {
		t.Fatalf("Runtimes count = %d, want 2", len(cfg.Runtimes))
	}
	if len(cfg.LSPs) != 2 {
		t.Fatalf("LSPs count = %d, want 2", len(cfg.LSPs))
	}

	// Spot-check: both tools present.
	tools := make(map[string]bool)
	for _, r := range cfg.Runtimes {
		tools[r.Tool] = true
	}
	if !tools["go"] {
		t.Error("missing runtime tool 'go'")
	}
	if !tools["node"] {
		t.Error("missing runtime tool 'node'")
	}

	// Spot-check: both LSP plugins present.
	plugins := make(map[string]bool)
	for _, l := range cfg.LSPs {
		plugins[l.Plugin] = true
	}
	if !plugins["gopls-lsp@claude-plugins-official"] {
		t.Error("missing LSP plugin 'gopls-lsp@claude-plugins-official'")
	}
	if !plugins["typescript-lsp@claude-plugins-official"] {
		t.Error("missing LSP plugin 'typescript-lsp@claude-plugins-official'")
	}
}

func TestMerge_AllStacks(t *testing.T) {
	allIDs := []stack.StackID{stack.Go, stack.Node, stack.Python, stack.Rust, stack.Ruby}
	cfg, err := Merge(allIDs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compute expected unique runtimes and LSPs from the registry.
	uniqueTools := make(map[string]bool)
	uniquePackages := make(map[string]bool)
	for _, id := range allIDs {
		s, ok := stack.Get(id)
		if !ok {
			t.Fatalf("stack.Get(%q) returned false", id)
		}
		uniqueTools[s.Runtime.Tool] = true
		uniquePackages[s.LSP.Package] = true
	}

	if len(cfg.Runtimes) != len(uniqueTools) {
		t.Errorf("Runtimes count = %d, want %d", len(cfg.Runtimes), len(uniqueTools))
	}
	if len(cfg.LSPs) != len(uniquePackages) {
		t.Errorf("LSPs count = %d, want %d", len(cfg.LSPs), len(uniquePackages))
	}

	// Verify no duplicates in runtimes.
	seenTools := make(map[string]bool)
	for _, r := range cfg.Runtimes {
		if seenTools[r.Tool] {
			t.Errorf("duplicate runtime tool %q", r.Tool)
		}
		seenTools[r.Tool] = true
	}

	// Verify no duplicates in LSPs.
	seenPackages := make(map[string]bool)
	for _, l := range cfg.LSPs {
		if seenPackages[l.Package] {
			t.Errorf("duplicate LSP package %q", l.Package)
		}
		seenPackages[l.Package] = true
	}
}

func TestMerge_EmptyStacks(t *testing.T) {
	cfg, err := Merge([]stack.StackID{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Runtimes and LSPs should be non-nil empty slices.
	if cfg.Runtimes == nil {
		t.Error("Runtimes is nil, want non-nil empty slice")
	}
	if len(cfg.Runtimes) != 0 {
		t.Errorf("Runtimes count = %d, want 0", len(cfg.Runtimes))
	}
	if cfg.LSPs == nil {
		t.Error("LSPs is nil, want non-nil empty slice")
	}
	if len(cfg.LSPs) != 0 {
		t.Errorf("LSPs count = %d, want 0", len(cfg.LSPs))
	}

	// Stacks should be non-nil empty slice.
	if cfg.Stacks == nil {
		t.Error("Stacks is nil, want non-nil empty slice")
	}
	if len(cfg.Stacks) != 0 {
		t.Errorf("Stacks count = %d, want 0", len(cfg.Stacks))
	}

	// Domains should still include always-on entries.
	wantDomains, fwErr := firewall.Merge(nil, nil)
	if fwErr != nil {
		t.Fatalf("firewall.Merge unexpected error: %v", fwErr)
	}
	if len(cfg.Domains.Static) != len(wantDomains.Static) {
		t.Errorf("Static domains count = %d, want %d", len(cfg.Domains.Static), len(wantDomains.Static))
	}
	if len(cfg.Domains.Dynamic) != len(wantDomains.Dynamic) {
		t.Errorf("Dynamic domains count = %d, want %d", len(cfg.Domains.Dynamic), len(wantDomains.Dynamic))
	}
}

func TestMerge_NilStacks(t *testing.T) {
	cfg, err := Merge(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Stacks == nil {
		t.Error("Stacks is nil, want non-nil empty slice")
	}
	if len(cfg.Stacks) != 0 {
		t.Errorf("Stacks count = %d, want 0", len(cfg.Stacks))
	}
	if cfg.Runtimes == nil {
		t.Error("Runtimes is nil, want non-nil empty slice")
	}
	if len(cfg.Runtimes) != 0 {
		t.Errorf("Runtimes count = %d, want 0", len(cfg.Runtimes))
	}
	if cfg.LSPs == nil {
		t.Error("LSPs is nil, want non-nil empty slice")
	}
	if len(cfg.LSPs) != 0 {
		t.Errorf("LSPs count = %d, want 0", len(cfg.LSPs))
	}

	// Domains should still include always-on entries.
	wantDomains, fwErr := firewall.Merge(nil, nil)
	if fwErr != nil {
		t.Fatalf("firewall.Merge unexpected error: %v", fwErr)
	}
	if len(cfg.Domains.Static) != len(wantDomains.Static) {
		t.Errorf("Static domains count = %d, want %d", len(cfg.Domains.Static), len(wantDomains.Static))
	}
	if len(cfg.Domains.Dynamic) != len(wantDomains.Dynamic) {
		t.Errorf("Dynamic domains count = %d, want %d", len(cfg.Domains.Dynamic), len(wantDomains.Dynamic))
	}
}

func TestMerge_UnknownStack(t *testing.T) {
	_, err := Merge([]stack.StackID{"elixir"}, nil)
	if err == nil {
		t.Fatal("expected error for unknown stack, got nil")
	}
	if !strings.Contains(err.Error(), "unknown stack") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "unknown stack")
	}
	if !strings.Contains(err.Error(), "elixir") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "elixir")
	}
}

func TestMerge_DuplicateStackIDs(t *testing.T) {
	single, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	double, err := Merge([]stack.StackID{stack.Go, stack.Go}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(single.Runtimes) != len(double.Runtimes) {
		t.Errorf("Runtimes: single=%d, double=%d", len(single.Runtimes), len(double.Runtimes))
	}
	if len(single.LSPs) != len(double.LSPs) {
		t.Errorf("LSPs: single=%d, double=%d", len(single.LSPs), len(double.LSPs))
	}
	if len(single.Stacks) != len(double.Stacks) {
		t.Errorf("Stacks: single=%d, double=%d", len(single.Stacks), len(double.Stacks))
	}
}

func TestMerge_SortedOutput(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Ruby, stack.Go, stack.Node}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stacks should be sorted alphabetically.
	if !slices.IsSortedFunc(cfg.Stacks, func(a, b stack.StackID) int {
		return strings.Compare(string(a), string(b))
	}) {
		t.Errorf("Stacks not sorted: %v", cfg.Stacks)
	}

	// Runtimes should be sorted by Tool.
	if !slices.IsSortedFunc(cfg.Runtimes, func(a, b stack.Runtime) int {
		return strings.Compare(a.Tool, b.Tool)
	}) {
		tools := make([]string, len(cfg.Runtimes))
		for i, r := range cfg.Runtimes {
			tools[i] = r.Tool
		}
		t.Errorf("Runtimes not sorted by Tool: %v", tools)
	}

	// LSPs should be sorted by Package.
	if !slices.IsSortedFunc(cfg.LSPs, func(a, b stack.LSP) int {
		return strings.Compare(a.Package, b.Package)
	}) {
		packages := make([]string, len(cfg.LSPs))
		for i, l := range cfg.LSPs {
			packages[i] = l.Package
		}
		t.Errorf("LSPs not sorted by Package: %v", packages)
	}
}

func TestMerge_UserExtraDomains(t *testing.T) {
	extras := []string{"custom.example.com", "another.example.com"}
	cfg, err := Merge([]stack.StackID{stack.Go}, extras)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// User extras should appear in Domains.Dynamic.
	dynamicNames := make(map[string]bool)
	for _, d := range cfg.Domains.Dynamic {
		dynamicNames[d.Name] = true
	}

	if !dynamicNames["custom.example.com"] {
		t.Error("missing user extra custom.example.com in Dynamic")
	}
	if !dynamicNames["another.example.com"] {
		t.Error("missing user extra another.example.com in Dynamic")
	}
}

func TestMerge_StacksFieldMatchesInput(t *testing.T) {
	input := []stack.StackID{stack.Node, stack.Go, stack.Go, stack.Node}
	cfg, err := Merge(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain exactly the deduplicated, sorted input stack IDs.
	want := []stack.StackID{stack.Go, stack.Node}
	if !slices.Equal(cfg.Stacks, want) {
		t.Errorf("Stacks = %v, want %v", cfg.Stacks, want)
	}
}

func TestMerge_RuntimesMatchRegistry(t *testing.T) {
	input := []stack.StackID{stack.Go, stack.Python, stack.Rust}
	cfg, err := Merge(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compute expected unique tools from the registry.
	expectedTools := make(map[string]bool)
	for _, id := range input {
		s, ok := stack.Get(id)
		if !ok {
			t.Fatalf("stack.Get(%q) returned false", id)
		}
		expectedTools[s.Runtime.Tool] = true
	}

	if len(cfg.Runtimes) != len(expectedTools) {
		t.Errorf("Runtimes count = %d, want %d", len(cfg.Runtimes), len(expectedTools))
	}

	// Verify each expected tool is present.
	resultTools := make(map[string]bool)
	for _, r := range cfg.Runtimes {
		resultTools[r.Tool] = true
	}
	for tool := range expectedTools {
		if !resultTools[tool] {
			t.Errorf("missing runtime tool %q", tool)
		}
	}
}

func TestMerge_SystemDeps_GoOnly(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Go}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.SystemDeps == nil {
		t.Error("SystemDeps is nil, want non-nil empty slice")
	}
	if len(cfg.SystemDeps) != 0 {
		t.Errorf("SystemDeps = %v, want empty (Go has no system deps)", cfg.SystemDeps)
	}
}

func TestMerge_SystemDeps_Ruby(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Ruby}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ruby, _ := stack.Get(stack.Ruby)
	if len(cfg.SystemDeps) != len(ruby.SystemDeps) {
		t.Errorf("SystemDeps count = %d, want %d", len(cfg.SystemDeps), len(ruby.SystemDeps))
	}

	// Spot-check specific Ruby deps.
	depsSet := make(map[string]bool)
	for _, d := range cfg.SystemDeps {
		depsSet[d] = true
	}
	if !depsSet["libssl-dev"] {
		t.Error("missing libssl-dev in SystemDeps for Ruby")
	}
	if !depsSet["libreadline-dev"] {
		t.Error("missing libreadline-dev in SystemDeps for Ruby")
	}
}

func TestMerge_SystemDeps_Deduplication(t *testing.T) {
	// Ruby and Python both declare libssl-dev; merged should contain it only once.
	cfg, err := Merge([]stack.StackID{stack.Ruby, stack.Python}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	count := 0
	for _, d := range cfg.SystemDeps {
		if d == "libssl-dev" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("libssl-dev appears %d times, want exactly 1 (deduplication)", count)
	}

	// Structural: count should match union of unique deps from both stacks.
	ruby, _ := stack.Get(stack.Ruby)
	python, _ := stack.Get(stack.Python)
	union := make(map[string]bool)
	for _, d := range ruby.SystemDeps {
		union[d] = true
	}
	for _, d := range python.SystemDeps {
		union[d] = true
	}
	if len(cfg.SystemDeps) != len(union) {
		t.Errorf("SystemDeps count = %d, want %d (union of Ruby+Python)", len(cfg.SystemDeps), len(union))
	}
}

func TestMerge_SystemDeps_Sorted(t *testing.T) {
	cfg, err := Merge([]stack.StackID{stack.Ruby, stack.Python}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !slices.IsSorted(cfg.SystemDeps) {
		t.Errorf("SystemDeps not sorted: %v", cfg.SystemDeps)
	}
}

func TestMerge_SystemDeps_Empty(t *testing.T) {
	cfg, err := Merge([]stack.StackID{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.SystemDeps == nil {
		t.Error("SystemDeps is nil, want non-nil empty slice")
	}
	if len(cfg.SystemDeps) != 0 {
		t.Errorf("SystemDeps count = %d, want 0", len(cfg.SystemDeps))
	}
}

func TestMerge_LSPsMatchRegistry(t *testing.T) {
	input := []stack.StackID{stack.Go, stack.Python, stack.Rust}
	cfg, err := Merge(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Compute expected unique packages from the registry.
	expectedPackages := make(map[string]bool)
	for _, id := range input {
		s, ok := stack.Get(id)
		if !ok {
			t.Fatalf("stack.Get(%q) returned false", id)
		}
		expectedPackages[s.LSP.Package] = true
	}

	if len(cfg.LSPs) != len(expectedPackages) {
		t.Errorf("LSPs count = %d, want %d", len(cfg.LSPs), len(expectedPackages))
	}

	// Verify each expected package is present.
	resultPackages := make(map[string]bool)
	for _, l := range cfg.LSPs {
		resultPackages[l.Package] = true
	}
	for pkg := range expectedPackages {
		if !resultPackages[pkg] {
			t.Errorf("missing LSP package %q", pkg)
		}
	}
}
