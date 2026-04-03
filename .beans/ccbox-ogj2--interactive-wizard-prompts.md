---
# ccbox-ogj2
title: Interactive wizard prompts
status: in-progress
type: task
priority: normal
created_at: 2026-04-02T10:34:37Z
updated_at: 2026-04-03T07:23:59Z
parent: ccbox-puuq
---

## Description
Implement the interactive wizard flow for `ccbox init` (default when no flags given):

1. **Scan & confirm stacks**: Show detected stacks, let user toggle on/off, offer to add undetected ones
2. **Extra domains**: Ask if user wants to add custom domains to the firewall allowlist
3. **Confirmation**: Show summary of what will be generated, confirm before writing

Use a Go TUI library (e.g., charmbracelet/huh or bubbletea) for the prompts. Should feel polished and modern.

## Implementation Plan

### Approach

Introduce a new `internal/wizard` package that encapsulates the interactive prompt flow using `charmbracelet/huh` (a high-level forms library built on bubbletea). The wizard collects user choices into a pure data struct (`wizard.Choices`) that the `cmd/init.go` RunE function consumes to drive the existing `render.Merge` pipeline. The wizard package has no file I/O -- it only gathers user input.

**Why `huh` over raw `bubbletea`?** `huh` provides pre-built form components (multi-select, text input, confirm) with accessibility support, consistent styling via lipgloss themes, and a declarative API. Building equivalent UX with raw bubbletea would require substantially more code for the same outcome. `huh` is maintained by the Charm team alongside bubbletea and is their recommended approach for form-style prompts.

**Why a new package instead of inline in `cmd/`?** Separating prompt logic from command wiring follows the existing architecture: `cmd/` orchestrates, internal packages own behavior. This also enables testing the wizard choices pipeline independently of Cobra.

### Key Design Decisions

1. **Testability via an interface.** Define a `Prompter` interface in `internal/wizard/` with a single method (`Run(detected []stack.StackID) (Choices, error)`). The production implementation uses `huh` forms. Tests can inject a fake `Prompter` that returns canned `Choices` without any terminal interaction. This follows the `fs.FS` testability pattern already used in `internal/detect/`.

2. **Wizard triggers when no `--stacks` flag is provided and stdin is a terminal.** When `--stacks` is given, the wizard is skipped entirely (current behavior preserved). When stdin is not a TTY (piped input, CI), skip the wizard and fall through to auto-detect-only behavior. This avoids breaking scripted usage before the `--non-interactive` flag bean (ccbox-nvf1) lands.

3. **Three-step wizard flow:**
   - Step 1: Multi-select of stacks (detected ones pre-checked, all available stacks listed)
   - Step 2: Text input for extra domains (comma-separated, with validation feedback)
   - Step 3: Confirmation summary showing stacks + domains, with proceed/cancel

4. **Boundary with sibling beans.** This bean does NOT implement `.ccbox.yml` writing (ccbox-mp0k) or the `--non-interactive` flag (ccbox-nvf1). The wizard outputs a `Choices` struct; downstream consumers decide what to do with it.

### Files to Create/Modify

- `internal/wizard/doc.go` -- Package documentation
- `internal/wizard/wizard.go` -- `Choices` struct, `Prompter` interface, `HuhPrompter` implementation
- `internal/wizard/wizard_test.go` -- Tests for Choices validation, summary formatting, TTY detection helper
- `cmd/init.go` -- Wire wizard into RunE: detect stacks, check TTY, run wizard or fall through, pass choices to Merge
- `cmd/init_test.go` -- Update existing tests, add tests for wizard integration using fake Prompter
- `go.mod` / `go.sum` -- Add `charmbracelet/huh` dependency (and transitive deps: bubbletea, lipgloss, etc.)

### Steps

#### 1. Add `charmbracelet/huh` dependency

Run `go get github.com/charmbracelet/huh@latest` to add the dependency. This pulls in bubbletea, lipgloss, and related Charm libraries as transitive dependencies.

#### 2. Create `internal/wizard/doc.go`

Package doc comment explaining that `wizard` implements the interactive prompt flow for `ccbox init`. It collects user choices (stack selection, extra domains) without performing any file I/O or template rendering.

#### 3. Create `internal/wizard/wizard.go`

Define the following types and functions:

```go
// Choices holds the user selections from the interactive wizard.
// It is a pure data struct with no behavior -- the cmd layer
// consumes it to drive render.Merge.
type Choices struct {
    Stacks       []stack.StackID
    ExtraDomains []string
}
```

```go
// Prompter abstracts the interactive prompt flow for testability.
// Production code uses HuhPrompter; tests inject a fake.
type Prompter interface {
    Run(detected []stack.StackID) (Choices, error)
}
```

```go
// HuhPrompter implements Prompter using charmbracelet/huh forms.
type HuhPrompter struct{}
```

The `HuhPrompter.Run` method builds a `huh.Form` with three groups:

**Group 1 -- Stack Selection:**
- `huh.MultiSelect` listing all stacks from `stack.All()`. Each option shows `stack.Name` (e.g., "Go", "Python") with `stack.ID` as the value. Detected stacks are pre-selected via the `Value` pointer initialized with the detected slice.
- Title: "Which stacks should be included?"
- Description: "Auto-detected stacks are pre-selected. Toggle to add or remove."

**Group 2 -- Extra Domains:**
- `huh.Text` (multi-line) for entering extra domains, one per line.
- Title: "Extra domains to allowlist (optional)"
- Description: "Enter additional domains for the firewall allowlist, one per line. Leave empty to skip."
- Validation function: split by newlines, trim whitespace, skip empty lines, call `firewall.ValidateDomain` on each. Return a combined error if any fail.

**Group 3 -- Confirmation:**
- `huh.Confirm` showing a summary of the selections.
- Title: "Generate .devcontainer/ with this configuration?"
- Description dynamically built: list selected stacks and extra domains.
- If the user declines, return a sentinel error (`ErrAborted`).

Define `ErrAborted = errors.New("wizard: user cancelled")` as a package-level sentinel.

After the form completes, parse the extra domains text into a `[]string` (split by newlines, trim, filter empty), populate `Choices`, and return.

#### 4. Create `internal/wizard/wizard_test.go`

Tests for the wizard package (all unit tests, no terminal interaction):

- **TestChoices_Empty**: Verify zero-value Choices has nil/empty slices.
- **TestParseDomains**: Test the internal domain parsing helper (newlines, whitespace, empty lines, duplicates). Extract the parsing logic into an unexported `parseDomains(text string) []string` function for testability.
- **TestParseDomains_Validation**: Verify that invalid domains (shell metacharacters, empty after trim) are caught by `firewall.ValidateDomain`.
- **TestErrAborted**: Verify the sentinel error is distinct and matchable via `errors.Is`.

Note: The `HuhPrompter` itself cannot be unit-tested without a terminal. Integration-level verification happens in `cmd/init_test.go` via the `Prompter` interface with a fake. The `huh` library handles its own rendering correctness.

#### 5. Modify `cmd/init.go` -- Wire the wizard

Restructure `newInitCmd().RunE` to support the wizard flow:

```
func newInitCmd() *cobra.Command {
    var stacks []string
    var domains []string

    cmd := &cobra.Command{
        ...
        RunE: func(cmd *cobra.Command, _ []string) error {
            dir, err := os.Getwd()
            ...

            var stackIDs []stack.StackID
            var extraDomains []string

            stacksFlagSet := cmd.Flags().Changed("stacks")

            if stacksFlagSet {
                // CLI flag path: parse --stacks directly (existing behavior)
                for _, s := range stacks {
                    stackIDs = append(stackIDs, stack.StackID(s))
                }
                extraDomains = domains
            } else {
                // Auto-detect stacks
                detected, err := detect.Detect(dir)
                if err != nil {
                    return fmt.Errorf("detect stacks: %w", err)
                }

                // Check if we should run the wizard
                if isTerminal(cmd.InOrStdin()) {
                    prompter := wizardPrompter(cmd)
                    choices, err := prompter.Run(detected)
                    if err != nil {
                        if errors.Is(err, wizard.ErrAborted) {
                            fmt.Fprintln(cmd.ErrOrStderr(), "Cancelled.")
                            return nil
                        }
                        return err
                    }
                    stackIDs = choices.Stacks
                    extraDomains = choices.ExtraDomains
                } else {
                    // Non-interactive fallback: use detected stacks as-is
                    stackIDs = detected
                    extraDomains = domains
                    if len(stackIDs) == 0 {
                        fmt.Fprintln(cmd.ErrOrStderr(), "No stacks detected. Use --stacks to specify manually.")
                        return nil
                    }
                }
            }

            // Existing Merge + render + write pipeline continues unchanged...
            cfg, err := render.Merge(stackIDs, extraDomains)
            ...
        },
    }
    ...
}
```

Key implementation details:

- **`isTerminal` helper**: An unexported function that checks if the given `io.Reader` is a `*os.File` with `term.IsTerminal(int(f.Fd()))`. Use `golang.org/x/term` (already a transitive dep of `huh`/bubbletea). This is a simple type assertion + syscall, not an interface -- terminal detection is inherently platform-specific.

- **`wizardPrompter` function variable**: A package-level `var wizardPrompter = func(cmd *cobra.Command) wizard.Prompter` that returns `&wizard.HuhPrompter{}` by default. Tests override this variable to inject a fake prompter. This follows the same pattern as `var rootCmd = newRootCmd()` -- a package-level variable with a test-friendly seam.

  Alternative considered: passing `Prompter` as a parameter through `newInitCmd()`. Rejected because it complicates the Cobra constructor pattern established in ADR-0001. The function variable approach is simpler and keeps the command constructor signature clean.

#### 6. Modify `cmd/init_test.go` -- Test the wizard integration

Add the following tests:

- **TestInitCommand_WizardFlow**: Override `wizardPrompter` to return a fake that returns canned Choices (e.g., stacks=["go"], domains=["api.example.com"]). Verify the .devcontainer/ is generated with the expected configuration. Restore the original `wizardPrompter` in cleanup.

- **TestInitCommand_WizardAborted**: Override `wizardPrompter` to return `wizard.ErrAborted`. Verify no .devcontainer/ directory is created, and the command exits cleanly (no error).

- **TestInitCommand_StacksFlagSkipsWizard**: Existing test (`TestInitCommand_WithStacksFlag`) already covers this. Add an assertion that `wizardPrompter` is never called when `--stacks` is set. Use a fake that calls `t.Fatal("wizard should not run")`.

- **TestInitCommand_NonTTYSkipsWizard**: Set `cmd.SetIn(strings.NewReader(""))` (not a terminal) and verify wizard is not invoked. This tests the `isTerminal` fallback path.

Existing tests (`TestInitCommand_GeneratesDevcontainer`, `TestInitCommand_WithStacksFlag`) continue to work unchanged because they either provide `--stacks` or run in a non-TTY test environment where `isTerminal` returns false.

#### 7. Write ADR for `charmbracelet/huh` dependency and wizard architecture

Create `decisions/0007-huh-forms-for-interactive-wizard.md` documenting:
- Why `huh` over raw bubbletea (declarative forms vs. manual state machine)
- Why `huh` over simpler prompt libraries like `survey` (survey is archived, huh is actively maintained)
- The `Prompter` interface pattern for testability
- The TTY detection strategy for wizard triggering

### Testing Strategy

1. **Unit tests in `internal/wizard/`**: Test domain parsing, validation integration, sentinel error. These are pure logic tests with no terminal dependency.

2. **Integration tests in `cmd/`**: Test the full init flow with a fake `Prompter`. Verify that wizard choices flow correctly through Merge to rendered output. Verify abort handling. Verify that `--stacks` and non-TTY paths skip the wizard.

3. **Manual smoke test**: Run `ccbox init` in a terminal, interact with the wizard, verify the generated .devcontainer/ looks correct. Run with `echo | ccbox init` to verify non-TTY fallback.

4. **What to verify:**
   - Detected stacks appear pre-selected in multi-select
   - User can deselect detected stacks and add undetected ones
   - Invalid domains show validation errors inline
   - Cancellation at confirmation produces no output
   - `--stacks` flag completely bypasses the wizard
   - Non-TTY stdin completely bypasses the wizard
   - Empty stack selection after wizard shows appropriate message
   - All existing tests continue to pass without modification

### Open Questions

1. **Multi-line text vs repeated single-line for domains.** The plan uses `huh.Text` (multi-line textarea) for entering extra domains. An alternative is a loop with `huh.Input` that asks "Add another domain?" after each entry. The multi-line approach is simpler to implement and test, but the loop approach may be more user-friendly for small numbers of domains. Decision: start with multi-line, iterate if user feedback suggests otherwise.

2. **Theme/styling.** The `huh` library supports themes (Charm, Dracula, Catppuccin, Base16, default). The plan uses the default theme, which looks good in most terminals. A custom theme could be added later to match ccbox branding, but this is cosmetic and out of scope for this bean.

## Checklist

- [ ] Tests written (TDD)
- [ ] No TODO/FIXME/HACK/XXX comments
- [ ] Lint passes
- [ ] Tests pass
- [ ] Branch pushed
- [ ] PR created
- [ ] Automated code review passed
- [ ] Review feedback worked in
- [ ] All other checklist items

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | done | 1 | 2026-04-03 |
| challenge | pending | | |
| implement | pending | | |
| pr | pending | | |
| review | pending | | |
| codify | pending | | |
