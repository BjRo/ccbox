package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bjro/ccbox/internal/detect"
	"github.com/bjro/ccbox/internal/render"
	"github.com/bjro/ccbox/internal/stack"
	"github.com/bjro/ccbox/internal/wizard"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// newInitCmd creates the init subcommand. The prompter parameter controls
// the interactive wizard: when non-nil it is used directly (test path),
// when nil the command checks for a TTY and instantiates HuhPrompter
// for real terminal sessions.
func newInitCmd(prompter wizard.Prompter) *cobra.Command {
	var stacks []string
	var domains []string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a devcontainer configuration",
		Long:  "Generate a .devcontainer/ directory with Dockerfile, firewall scripts, Claude Code settings, and documentation.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}

			var stackIDs []stack.StackID
			var extraDomains []string

			stacksFlagSet := cmd.Flags().Changed("stacks")

			if stacksFlagSet {
				// CLI flag path: parse --stacks directly (existing behavior).
				for _, s := range stacks {
					stackIDs = append(stackIDs, stack.StackID(s))
				}
				extraDomains = domains
			} else {
				// Auto-detect stacks.
				detected, detectErr := detect.Detect(dir)
				if detectErr != nil {
					return fmt.Errorf("detect stacks: %w", detectErr)
				}

				// Determine whether to run the wizard.
				// When a prompter is explicitly provided (test injection),
				// always use it. Otherwise check for a real TTY and
				// instantiate HuhPrompter for terminal sessions.
				if prompter == nil && isTerminal(cmd.InOrStdin()) {
					prompter = &wizard.HuhPrompter{}
				}
				if prompter != nil {
					choices, wizErr := prompter.Run(detected)
					if wizErr != nil {
						if errors.Is(wizErr, wizard.ErrAborted) {
							_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Cancelled.")
							return nil
						}
						return wizErr
					}
					stackIDs = choices.Stacks
					extraDomains = choices.ExtraDomains
				} else {
					// Non-interactive fallback: use detected stacks as-is.
					stackIDs = detected
					extraDomains = domains
				}

				// Empty-stack guard (challenge finding 2).
				if len(stackIDs) == 0 {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "No stacks selected. Use --stacks to specify manually.")
					return nil
				}
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Stacks: %v\n", stackIDs)

			// Merge configuration.
			cfg, err := render.Merge(stackIDs, extraDomains)
			if err != nil {
				return fmt.Errorf("merge config: %w", err)
			}

			// Render all templates.
			dockerfile, err := render.Dockerfile(cfg)
			if err != nil {
				return err
			}

			var devcontainerBuf bytes.Buffer
			if err := render.DevContainer(&devcontainerBuf, cfg); err != nil {
				return err
			}

			fw, err := render.RenderFirewall(cfg)
			if err != nil {
				return err
			}

			cl, err := render.RenderClaude(cfg)
			if err != nil {
				return err
			}

			readme, err := render.README(cfg)
			if err != nil {
				return err
			}

			// Write .devcontainer/ directory.
			outDir := filepath.Join(dir, ".devcontainer")
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return fmt.Errorf("create .devcontainer: %w", err)
			}

			files := map[string][]byte{
				"Dockerfile":                []byte(dockerfile),
				"devcontainer.json":         devcontainerBuf.Bytes(),
				"init-firewall.sh":          fw.InitFirewall,
				"warmup-dns.sh":             fw.WarmupDNS,
				"dynamic-domains.conf":      fw.DynamicDomains,
				"claude-user-settings.json": cl.UserSettings,
				"sync-claude-settings.sh":   cl.SyncSettings,
				"README.md":                 []byte(readme),
			}

			for name, content := range files {
				path := filepath.Join(outDir, name)
				if err := os.WriteFile(path, content, 0o644); err != nil {
					return fmt.Errorf("write %s: %w", name, err)
				}
			}

			// Make shell scripts executable.
			for _, name := range []string{"init-firewall.sh", "warmup-dns.sh", "sync-claude-settings.sh"} {
				path := filepath.Join(outDir, name)
				if err := os.Chmod(path, 0o755); err != nil {
					return fmt.Errorf("chmod %s: %w", name, err)
				}
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Generated .devcontainer/ with %d files\n", len(files))
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&stacks, "stacks", nil, "Comma-separated stack IDs (e.g., go,node). Auto-detects if omitted.")
	cmd.Flags().StringSliceVar(&domains, "domains", nil, "Extra domains to allowlist (e.g., api.example.com)")

	return cmd
}

// isTerminal reports whether r is a terminal file descriptor.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}
