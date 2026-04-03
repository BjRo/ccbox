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


## Challenge Report

**Scope: BIG CHANGE** (6 files created/modified + go.mod/go.sum)

### Scope Assessment

| Metric | Value | Threshold |
|--------|-------|-----------|
| Files | 8 (including go.mod/go.sum) | >15 = recommend split |

The scope is manageable. New package + cmd wiring + dependency addition is a natural unit of work.

### Findings

#### Go Engineer

> **Finding 1: Package-level `wizardPrompter` function variable breaks test parallelism** (severity: WARNING)
>
> Step 5 proposes `var wizardPrompter = func(cmd *cobra.Command) wizard.Prompter` as the test seam. Tests override this package-level variable and restore it in cleanup. This is shared mutable state: tests that override `wizardPrompter` cannot run with `t.Parallel()`. The existing `TestInitCommand_GeneratesDevcontainer` test (which does NOT set `--stacks`) will also be affected -- it currently succeeds because `isTerminal` returns false in test, but if any test overwrites `wizardPrompter` concurrently, behavior is undefined.
>
> The plan acknowledges this tradeoff but dismisses the alternative (passing `Prompter` through `newInitCmd`) too quickly. The Cobra constructor pattern from ADR-0001 is about unexported constructors returning fresh command trees -- it does not prohibit parameters. Adding `func newInitCmd(opts ...InitOption)` or simply `func newInitCmd(prompter wizard.Prompter)` is a one-line signature change that eliminates shared state entirely.
>
> **Option A (recommended):** Accept a `wizard.Prompter` parameter in `newInitCmd()`. Production code passes `nil` (meaning "use default HuhPrompter"), and the function falls back internally: `if prompter == nil { prompter = &wizard.HuhPrompter{} }`. Tests pass their fake directly. This is a trivial change to `newRootCmd()` (pass `nil`) and keeps all tests parallelizable.
>
> **Option B:** Keep the function variable but document that all wizard-related tests in `cmd/init_test.go` must NOT use `t.Parallel()`. Add a comment at the top of the test file explaining the constraint.
>
> **Option C:** Use a context value to carry the prompter. Overly indirect -- not recommended.

> **Finding 2: Empty stack selection after wizard has no guard** (severity: WARNING)
>
> Step 5's pseudocode shows that when the wizard returns successfully, `stackIDs = choices.Stacks` is used directly. If the user deselects all stacks in the multi-select and confirms, `choices.Stacks` will be an empty slice. This flows into `render.Merge([]stack.StackID{}, extraDomains)` which returns an empty `GenerationConfig`. The downstream render calls will produce degenerate output (e.g., a Dockerfile with no runtimes). The plan's testing checklist item "Empty stack selection after wizard shows appropriate message" (line 235) correctly identifies this risk but the implementation pseudocode has no corresponding guard.
>
> The existing non-TTY fallback path has a `len(stackIDs) == 0` check (line 174), but the wizard success path (line 168) does not.
>
> **Option A (recommended):** After `stackIDs = choices.Stacks`, add the same empty-stack guard that already exists in the non-TTY fallback path. Print a message and return nil.
>
> **Option B:** Enforce minimum one stack selection inside `HuhPrompter.Run` using `huh.MultiSelect`'s `Validate` callback. Return an error if the selection is empty. This is better UX (inline feedback) but should be paired with Option A as defense-in-depth.

> **Finding 3: Confirmation group cannot access form state from prior groups** (severity: WARNING)
>
> Step 3, Group 3 says the confirmation description is "dynamically built" to show selected stacks and extra domains. In `huh`, a `Form` with multiple `Group`s runs each group sequentially, but the description string for Group 3 must be set at form construction time. The form does not re-evaluate description strings between groups. This means the confirmation summary will show the initially-detected stacks and empty domain text -- NOT the user's actual selections from Groups 1 and 2.
>
> This is a fundamental `huh` API constraint. The description is a static string, not a function.
>
> **Option A (recommended):** Split into two sequential forms. Form 1 has Groups 1 and 2 (stack selection + domain input). After Form 1 completes, parse the results, build the summary string, then run Form 2 (a single `huh.Confirm` with the dynamic description). This is a minor structural change that makes the confirmation accurate.
>
> **Option B:** Use `huh`'s `WithDescription` on the confirm with a pointer-based dynamic value via a closure that reads from the multi-select's bound variable. This works for stacks (bound to a `*[]stack.StackID`) but is fragile for the domain text which needs parsing.
>
> **Option C:** Drop the dynamic confirmation summary and use a static message like "Proceed with the selections above?". Less informative but simpler.

> **Finding 4: `--domains` flag is silently ignored during wizard flow** (severity: SUGGESTION)
>
> In the plan's Step 5 pseudocode, when the wizard runs (TTY + no `--stacks` flag), the `domains` CLI flag variable is never consulted. The wizard collects its own extra domains via the text input. If a user runs `ccbox init --domains api.example.com` without `--stacks`, the wizard fires and the `--domains` value is silently dropped.
>
> This is arguably correct behavior (the wizard supersedes flags), but it could surprise users. The sibling bean ccbox-nvf1 will add `--non-interactive` to address this fully, but until then there is a gap.
>
> **Suggestion:** When `--domains` is set but `--stacks` is not, either (a) pre-populate the domain text field in the wizard with the flag values, or (b) skip the wizard domain step and use the flag values. Option (a) is the better UX and straightforward to implement -- initialize the `huh.Text` value pointer with the joined flag domains.

### Verdict

**APPROVED** -- with Findings 1-3 addressed during implementation.

The overall architecture is sound. The `Prompter` interface, the `Choices` data struct, the new `internal/wizard` package, and the TTY detection strategy are all well-considered. The `charmbracelet/huh` dependency is the right choice for this use case -- it is actively maintained, it matches the form-style UX the bean describes, and the alternative (`survey`) is archived.

Finding 1 (function variable) is a WARNING because it introduces shared mutable test state into a codebase that currently has none. The fix is simple. Finding 2 (empty stack guard) is a WARNING because it is a missing edge case that the plan itself identified but did not implement. Finding 3 (confirmation summary) is a WARNING because it will produce incorrect output if implemented as described -- the `huh` API does not support dynamic descriptions across groups.

Finding 4 is a SUGGESTION that can be deferred to ccbox-nvf1 if preferred.

## Checklist

- [x] Tests written (TDD)
- [x] No TODO/FIXME/HACK/XXX comments
- [x] Lint passes
- [x] Tests pass
- [x] Branch pushed
- [x] PR created
- [x] Automated code review passed
- [x] Review feedback worked in
- [ ] All other checklist items

## Pipeline State

| Phase | Status | Iteration | Timestamp |
|-------|--------|-----------|-----------|
| refine | done | 1 | 2026-04-03 |
| challenge | done | 1 | 2026-04-03 |
| implement | done | 1 | 2026-04-03 |
| pr | done | 1 | 2026-04-03 |
| review | done | 2 | 2026-04-03 |
| codify | pending | | |
