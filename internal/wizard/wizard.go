package wizard

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bjro/ccbox/internal/firewall"
	"github.com/bjro/ccbox/internal/stack"
	"github.com/charmbracelet/huh"
)

// ErrAborted is returned when the user cancels the wizard at the
// confirmation step.
var ErrAborted = errors.New("wizard: user cancelled")

// Choices holds the user selections from the interactive wizard.
// It is a pure data struct with no behavior -- the cmd layer
// consumes it to drive render.Merge.
type Choices struct {
	Stacks       []stack.StackID
	ExtraDomains []string
}

// Prompter abstracts the interactive prompt flow for testability.
// Production code uses HuhPrompter; tests inject a fake.
type Prompter interface {
	Run(detected []stack.StackID) (Choices, error)
}

// HuhPrompter implements Prompter using charmbracelet/huh forms.
type HuhPrompter struct{}

// Run presents the interactive wizard and returns the user's choices.
//
// The wizard uses two sequential forms per challenge finding 3:
// Form 1 collects stacks and extra domains. After Form 1 completes,
// a dynamic summary is built and Form 2 presents a confirmation prompt.
func (h *HuhPrompter) Run(detected []stack.StackID) (Choices, error) {
	// Build options from the stack registry.
	allStacks := stack.All()
	options := make([]huh.Option[stack.StackID], 0, len(allStacks))
	for _, s := range allStacks {
		options = append(options, huh.NewOption(s.Name, s.ID))
	}

	// Initialize selected stacks with detected ones.
	selected := make([]stack.StackID, len(detected))
	copy(selected, detected)

	var domainsText string

	// Form 1: Stack selection + extra domains.
	form1 := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[stack.StackID]().
				Title("Which stacks should be included?").
				Description("Auto-detected stacks are pre-selected. Toggle to add or remove.").
				Options(options...).
				Value(&selected).
				Validate(func(s []stack.StackID) error {
					if len(s) == 0 {
						return errors.New("at least one stack must be selected")
					}
					return nil
				}),
			huh.NewText().
				Title("Extra domains to allowlist (optional)").
				Description("Enter additional domains for the firewall allowlist, one per line. Leave empty to skip.").
				Value(&domainsText).
				Validate(func(s string) error {
					return validateDomainsText(s)
				}),
		),
	)

	if err := form1.Run(); err != nil {
		return Choices{}, err
	}

	// Parse domains from the text input.
	extraDomains := parseDomains(domainsText)

	// Build the dynamic summary for confirmation.
	summary := buildSummary(selected, extraDomains)

	// Form 2: Confirmation with dynamic summary (challenge finding 3).
	var confirmed bool
	form2 := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Generate .devcontainer/ with this configuration?").
				Description(summary).
				Value(&confirmed),
		),
	)

	if err := form2.Run(); err != nil {
		return Choices{}, err
	}

	if !confirmed {
		return Choices{}, ErrAborted
	}

	return Choices{
		Stacks:       selected,
		ExtraDomains: extraDomains,
	}, nil
}

// parseDomains splits a newline-separated string into individual domain
// names, trimming whitespace and filtering empty lines. Duplicate domains
// are removed while preserving order.
func parseDomains(text string) []string {
	if text == "" {
		return []string{}
	}

	lines := strings.Split(text, "\n")
	seen := make(map[string]bool)
	var domains []string

	for _, line := range lines {
		d := strings.TrimSpace(line)
		if d == "" {
			continue
		}
		if seen[d] {
			continue
		}
		seen[d] = true
		domains = append(domains, d)
	}

	if domains == nil {
		return []string{}
	}
	return domains
}

// validateDomainsText validates that all non-empty lines in the text are
// well-formed DNS hostnames using firewall.ValidateDomain.
func validateDomainsText(text string) error {
	if text == "" {
		return nil
	}

	lines := strings.Split(text, "\n")
	var errs []string

	for _, line := range lines {
		d := strings.TrimSpace(line)
		if d == "" {
			continue
		}
		if err := firewall.ValidateDomain(d); err != nil {
			errs = append(errs, err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// buildSummary creates a human-readable summary of the selected stacks
// and extra domains for the confirmation prompt.
func buildSummary(stacks []stack.StackID, extraDomains []string) string {
	var b strings.Builder

	b.WriteString("Stacks: ")
	names := make([]string, 0, len(stacks))
	for _, id := range stacks {
		s, ok := stack.Get(id)
		if ok {
			names = append(names, s.Name)
		} else {
			names = append(names, string(id))
		}
	}
	b.WriteString(strings.Join(names, ", "))

	if len(extraDomains) > 0 {
		b.WriteString("\nExtra domains: ")
		b.WriteString(strings.Join(extraDomains, ", "))
	}

	return b.String()
}
