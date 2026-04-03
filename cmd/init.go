package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjro/ccbox/internal/config"
	"github.com/bjro/ccbox/internal/detect"
	"github.com/bjro/ccbox/internal/render"
	"github.com/bjro/ccbox/internal/stack"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var stacks []string
	var domains []string
	var dir string
	var nonInteractive bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a devcontainer configuration",
		Long:  "Generate a .devcontainer/ directory with Dockerfile, firewall scripts, Claude Code settings, and documentation.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Resolve target directory.
			targetDir, err := resolveDir(dir)
			if err != nil {
				return err
			}

			// Trim and filter flag values.
			stacks = trimAndFilter(stacks)
			domains = trimAndFilter(domains)

			// Determine stacks: from flag or auto-detect.
			var stackIDs []stack.StackID
			if len(stacks) > 0 {
				// Validate stack IDs against the registry.
				if err := validateStackIDs(stacks); err != nil {
					return err
				}
				for _, s := range stacks {
					stackIDs = append(stackIDs, stack.StackID(s))
				}
			} else {
				detected, err := detect.Detect(targetDir)
				if err != nil {
					return fmt.Errorf("detect stacks: %w", err)
				}
				stackIDs = detected
				if len(stackIDs) == 0 {
					return fmt.Errorf("no stacks detected; use --stack to specify manually")
				}
			}

			// Suppress unused variable warning for nonInteractive.
			// The flag is accepted to establish the API contract for the future wizard (ccbox-ogj2).
			_ = nonInteractive

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
			outDir := filepath.Join(targetDir, ".devcontainer")
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

			// Write .ccbox.yml to the project root.
			ccboxCfg := config.Config{
				Version:      1,
				Stacks:       make([]string, len(cfg.Stacks)),
				ExtraDomains: domains,
				GeneratedAt:  time.Now().UTC(),
				CcboxVersion: version,
			}
			for i, id := range cfg.Stacks {
				ccboxCfg.Stacks[i] = string(id)
			}
			if ccboxCfg.ExtraDomains == nil {
				ccboxCfg.ExtraDomains = []string{}
			}

			var ccboxBuf bytes.Buffer
			if err := config.Write(&ccboxBuf, ccboxCfg); err != nil {
				return fmt.Errorf("write %s: %w", config.Filename, err)
			}
			if err := os.WriteFile(filepath.Join(dir, config.Filename), ccboxBuf.Bytes(), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", config.Filename, err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Generated .devcontainer/ with %d files and %s\n", len(files), config.Filename)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&stacks, "stack", nil, "Comma-separated stack IDs (e.g., go,node). Auto-detects if omitted.")
	cmd.Flags().StringSliceVar(&domains, "extra-domains", nil, "Additional domains to allowlist beyond per-stack defaults (e.g., api.example.com)")
	cmd.Flags().StringVar(&dir, "dir", "", "Target directory (default: current directory)")
	cmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "y", false, "Skip all prompts, use detected stacks and defaults")

	return cmd
}

// resolveDir resolves the target directory from the --dir flag value.
// If dir is empty, it falls back to the current working directory.
// It validates that the resolved path exists and is a directory.
func resolveDir(dir string) (string, error) {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
		return wd, nil
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(absDir)
	if err != nil {
		return "", fmt.Errorf("--dir %s: %w", dir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("--dir %s: not a directory", dir)
	}

	return absDir, nil
}

// validateStackIDs checks that each stack ID is known in the registry.
func validateStackIDs(stacks []string) error {
	validIDs := stack.IDs()
	validSet := make(map[stack.StackID]bool, len(validIDs))
	for _, id := range validIDs {
		validSet[id] = true
	}

	for _, s := range stacks {
		if !validSet[stack.StackID(s)] {
			validStrings := make([]string, len(validIDs))
			for i, id := range validIDs {
				validStrings[i] = string(id)
			}
			return fmt.Errorf("unknown stack %q; valid stacks: %s", s, strings.Join(validStrings, ", "))
		}
	}
	return nil
}

// trimAndFilter trims whitespace from each value and filters out empty strings.
func trimAndFilter(values []string) []string {
	var result []string
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}
