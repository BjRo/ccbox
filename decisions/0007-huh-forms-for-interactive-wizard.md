# ADR-0007: charmbracelet/huh forms for interactive wizard

- **Date**: 2026-04-03
- **Status**: Accepted
- **Bean**: agentbox-ogj2

## Context

The `agentbox init` command needs an interactive wizard that collects user choices (stack selection, extra domains, confirmation) when run in a terminal without `--stacks` flags. This requires a TUI forms library for Go.

Key requirements:
- Multi-select with pre-checked detected stacks
- Multi-line text input with inline validation
- Confirmation prompt with dynamic summary
- Accessible, polished terminal UX
- Testable without a real terminal

## Decision

### Library choice: charmbracelet/huh

We chose [charmbracelet/huh](https://github.com/charmbracelet/huh) over the alternatives:

- **Raw bubbletea**: Would require building form components (multi-select, text input, confirm) from scratch. `huh` provides these out of the box with consistent styling.
- **AlecAivazis/survey**: Archived (no longer maintained). `huh` is actively maintained by the Charm team.
- **manifoldco/promptui**: Limited component set, no multi-select, less active maintenance.

`huh` is built on bubbletea and lipgloss, provides a declarative API for form-style prompts, and handles accessibility (screen readers, keyboard navigation) automatically.

### Prompter interface for testability

The `internal/wizard` package defines a `Prompter` interface with a single `Run(detected []stack.StackID) (Choices, error)` method. Production code uses `HuhPrompter` which drives real terminal forms. Tests inject a fake `Prompter` that returns canned `Choices` without any terminal interaction.

This follows the same `fs.FS` testability pattern used in `internal/detect/`: define an interface for the external dependency, accept it as a parameter, and let tests substitute a fake.

### Parameter injection over package-level variable

The `Prompter` is passed as a parameter through `newRootCmd(prompter) -> newInitCmd(prompter)` rather than stored as a package-level `var wizardPrompter`. This eliminates shared mutable state and keeps all tests parallelizable. The Cobra constructor pattern from ADR-0001 does not prohibit parameters.

Production code passes `nil` to `newRootCmd`, and `newInitCmd` falls back to `&wizard.HuhPrompter{}` when the prompter is nil and stdin is a terminal.

### Two-form architecture

The wizard uses two sequential `huh.Form` instances rather than a single form with three groups. This is necessary because `huh` evaluates group descriptions at form construction time, not between groups. The confirmation step needs to display a dynamic summary built from the user's stack and domain selections, which are only known after the first form completes.

### TTY detection strategy

The `isTerminal` helper uses `golang.org/x/term.IsTerminal` on the command's stdin file descriptor. When stdin is not a terminal (piped input, CI), the wizard is skipped entirely and the command falls through to auto-detect-only behavior. This avoids breaking scripted usage.

## Consequences

- **New dependency**: `charmbracelet/huh` v1.0.0 and its transitive dependencies (bubbletea, lipgloss, etc.) are added to `go.mod`. This increases binary size but is acceptable for a CLI tool.
- **`golang.org/x/term`** is added as a direct dependency for TTY detection. It was already a transitive dependency of `huh`/bubbletea.
- **Testing**: The `HuhPrompter` itself cannot be unit-tested without a terminal. Testing relies on the `Prompter` interface with fakes in `cmd/init_test.go` and manual smoke testing for the real terminal experience.
- **Future work**: The `--non-interactive` flag (agentbox-nvf1) will provide an explicit opt-out from the wizard, complementing the implicit TTY detection.
