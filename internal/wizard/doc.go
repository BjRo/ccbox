// Package wizard implements the interactive prompt flow for ccbox init.
//
// It collects user choices (stack selection, extra domains) using
// charmbracelet/huh forms without performing any file I/O or template
// rendering. The resulting [Choices] struct is consumed by the cmd layer
// to drive the render.Merge pipeline.
//
// Testability is achieved through the [Prompter] interface: production code
// uses [HuhPrompter] which drives real terminal forms, while tests inject
// a fake that returns canned choices without any terminal interaction.
package wizard
