# ADR-0001: Cobra constructor pattern for test isolation

- **Date**: 2026-04-02
- **Status**: Accepted
- **Bean**: ccbox-5333

## Context

Cobra CLI applications commonly register commands via package-level `init()` functions and a shared `rootCmd` variable. This creates problems for testing: tests share mutable state, require careful teardown, and cannot run in parallel safely.

## Decision

Every command is constructed by an unexported function (`newRootCmd()`, `newInitCmd()`, etc.) that returns a fresh `*cobra.Command`. The root constructor calls all sub-command constructors and wires the tree. Tests call `newRootCmd()` per test case to get a fully isolated command tree.

The single production instance is `var rootCmd = newRootCmd()` at package level. No `init()` functions are used for command registration.

Tests live in the internal `package cmd` (not `package cmd_test`) to access unexported constructors.

## Consequences

- Every test gets a clean command tree with no shared state.
- Adding a new command means: write `newXxxCmd()`, call it from `newRootCmd()`, test via `newRootCmd()`.
- Integration/black-box tests that only exercise the public `Execute()` API belong in a separate test package or test binary.
- Slightly more wiring code than `init()`-based registration, but the construction flow is explicit and traceable.
