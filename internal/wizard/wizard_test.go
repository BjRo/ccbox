package wizard

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bjro/agentbox/internal/stack"
)

func TestChoices_ZeroValue(t *testing.T) {
	t.Parallel()
	var c Choices
	if c.Stacks != nil {
		t.Error("zero-value Choices.Stacks should be nil")
	}
	if c.ExtraDomains != nil {
		t.Error("zero-value Choices.ExtraDomains should be nil")
	}
}

func TestParseDomains_EmptyInput(t *testing.T) {
	t.Parallel()
	got := parseDomains("")
	if len(got) != 0 {
		t.Errorf("parseDomains(\"\") = %v, want empty", got)
	}
}

func TestParseDomains_SingleDomain(t *testing.T) {
	t.Parallel()
	got := parseDomains("api.example.com")
	if len(got) != 1 || got[0] != "api.example.com" {
		t.Errorf("parseDomains(\"api.example.com\") = %v, want [api.example.com]", got)
	}
}

func TestParseDomains_MultipleLines(t *testing.T) {
	t.Parallel()
	got := parseDomains("api.example.com\ncdn.example.com\nstatic.example.com")
	want := []string{"api.example.com", "cdn.example.com", "static.example.com"}
	if len(got) != len(want) {
		t.Fatalf("parseDomains got %d entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("parseDomains[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestParseDomains_TrimsWhitespace(t *testing.T) {
	t.Parallel()
	got := parseDomains("  api.example.com  \n  cdn.example.com  ")
	want := []string{"api.example.com", "cdn.example.com"}
	if len(got) != len(want) {
		t.Fatalf("parseDomains got %d entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("parseDomains[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestParseDomains_SkipsEmptyLines(t *testing.T) {
	t.Parallel()
	got := parseDomains("api.example.com\n\n\ncdn.example.com\n\n")
	want := []string{"api.example.com", "cdn.example.com"}
	if len(got) != len(want) {
		t.Fatalf("parseDomains got %d entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("parseDomains[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestParseDomains_DeduplicatesDomains(t *testing.T) {
	t.Parallel()
	got := parseDomains("api.example.com\napi.example.com\ncdn.example.com")
	want := []string{"api.example.com", "cdn.example.com"}
	if len(got) != len(want) {
		t.Fatalf("parseDomains got %d entries, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("parseDomains[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestValidateDomainsText_ValidInput(t *testing.T) {
	t.Parallel()
	err := validateDomainsText("api.example.com\ncdn.example.com")
	if err != nil {
		t.Errorf("validateDomainsText valid input: %v", err)
	}
}

func TestValidateDomainsText_EmptyInput(t *testing.T) {
	t.Parallel()
	err := validateDomainsText("")
	if err != nil {
		t.Errorf("validateDomainsText empty input: %v", err)
	}
}

func TestValidateDomainsText_InvalidDomain(t *testing.T) {
	t.Parallel()
	err := validateDomainsText("api.example.com\n-invalid-.com")
	if err == nil {
		t.Error("validateDomainsText should return error for invalid domain")
	}
}

func TestValidateDomainsText_ShellMetacharacters(t *testing.T) {
	t.Parallel()
	err := validateDomainsText("$(evil).com")
	if err == nil {
		t.Error("validateDomainsText should reject shell metacharacters")
	}
}

func TestErrAborted_IsSentinel(t *testing.T) {
	t.Parallel()
	if !errors.Is(ErrAborted, ErrAborted) {
		t.Error("ErrAborted should match itself via errors.Is")
	}
}

func TestErrAborted_DistinctFromOtherErrors(t *testing.T) {
	t.Parallel()
	other := errors.New("some other error")
	if errors.Is(other, ErrAborted) {
		t.Error("other error should not match ErrAborted")
	}
}

func TestErrAborted_WrappedIsMatchable(t *testing.T) {
	t.Parallel()
	wrapped := fmt.Errorf("wrapped: %w", ErrAborted)
	if !errors.Is(wrapped, ErrAborted) {
		t.Error("wrapped ErrAborted should match via errors.Is")
	}
}

func TestBuildSummary_StacksOnly(t *testing.T) {
	t.Parallel()
	stacks := []stack.StackID{stack.Go, stack.Node}
	summary := buildSummary(stacks, nil)
	if summary == "" {
		t.Fatal("buildSummary should not return empty string")
	}
	// Should contain stack names
	if !strings.Contains(summary, "Go") {
		t.Error("summary should contain 'Go'")
	}
	if !strings.Contains(summary, "Node/TypeScript") {
		t.Error("summary should contain 'Node/TypeScript'")
	}
}

func TestBuildSummary_WithDomains(t *testing.T) {
	t.Parallel()
	stacks := []stack.StackID{stack.Go}
	domains := []string{"api.example.com", "cdn.example.com"}
	summary := buildSummary(stacks, domains)
	if !strings.Contains(summary, "api.example.com") {
		t.Error("summary should contain 'api.example.com'")
	}
	if !strings.Contains(summary, "cdn.example.com") {
		t.Error("summary should contain 'cdn.example.com'")
	}
}

func TestBuildSummary_NoDomains(t *testing.T) {
	t.Parallel()
	stacks := []stack.StackID{stack.Go}
	summary := buildSummary(stacks, nil)
	if strings.Contains(summary, "Extra domains") {
		t.Error("summary should not mention extra domains when none provided")
	}
}

func TestPrompterInterface_FakeImplementation(t *testing.T) {
	t.Parallel()
	// Verify that a fake can satisfy the Prompter interface.
	fake := &fakePrompter{
		choices: Choices{
			Stacks:       []stack.StackID{stack.Go},
			ExtraDomains: []string{"api.example.com"},
		},
	}
	var p Prompter = fake
	choices, err := p.Run([]stack.StackID{stack.Go})
	if err != nil {
		t.Fatalf("fake prompter: %v", err)
	}
	if len(choices.Stacks) != 1 || choices.Stacks[0] != stack.Go {
		t.Errorf("stacks = %v, want [go]", choices.Stacks)
	}
	if len(choices.ExtraDomains) != 1 || choices.ExtraDomains[0] != "api.example.com" {
		t.Errorf("domains = %v, want [api.example.com]", choices.ExtraDomains)
	}
}

// fakePrompter is a test double for Prompter.
type fakePrompter struct {
	choices Choices
	err     error
}

func (f *fakePrompter) Run(_ []stack.StackID) (Choices, error) {
	return f.choices, f.err
}
