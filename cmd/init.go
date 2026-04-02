package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"github.com/bjro/ccbox/internal/detect"
	"github.com/bjro/ccbox/internal/render"
	"github.com/bjro/ccbox/internal/stack"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
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

			// Determine stacks: from flag or auto-detect.
			var stackIDs []stack.StackID
			if len(stacks) > 0 {
				for _, s := range stacks {
					stackIDs = append(stackIDs, stack.StackID(s))
				}
			} else {
				detected, err := detect.Detect(dir)
				if err != nil {
					return fmt.Errorf("detect stacks: %w", err)
				}
				stackIDs = detected
				if len(stackIDs) == 0 {
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "No stacks detected. Use --stacks to specify manually.")
					return nil
				}
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Stacks: %v\n", stackIDs)

			// Merge configuration.
			cfg, err := render.Merge(stackIDs, domains)
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
				"Dockerfile":                  []byte(dockerfile),
				"devcontainer.json":           devcontainerBuf.Bytes(),
				"init-firewall.sh":            fw.InitFirewall,
				"warmup-dns.sh":               fw.WarmupDNS,
				"dynamic-domains.conf":        fw.DynamicDomains,
				"claude-user-settings.json":   cl.UserSettings,
				"sync-claude-settings.sh":     cl.SyncSettings,
				"README.md":                   []byte(readme),
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

