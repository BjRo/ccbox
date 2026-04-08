package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjro/agentbox/internal/config"
	"github.com/bjro/agentbox/internal/detect"
	"github.com/bjro/agentbox/internal/render"
	"github.com/bjro/agentbox/internal/stack"
	"github.com/bjro/agentbox/internal/wizard"
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
	var dir string
	var nonInteractive bool
	var runtimeVersions []string

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

			// Fail fast if .devcontainer/ already exists (file or directory).
			outDir := filepath.Join(targetDir, ".devcontainer")
			if _, statErr := os.Stat(outDir); statErr == nil {
				return fmt.Errorf(".devcontainer/ already exists in %s; run 'agentbox update' to regenerate, or remove it first", targetDir)
			}

			// Trim and filter flag values.
			stacks = trimAndFilter(stacks)
			domains = trimAndFilter(domains)

			var stackIDs []stack.StackID
			var extraDomains []string
			var choices wizard.Choices

			stackFlagSet := len(stacks) > 0

			if stackFlagSet {
				// CLI flag path: validate and parse --stack directly.
				if err := validateStackIDs(stacks); err != nil {
					return err
				}
				for _, s := range stacks {
					stackIDs = append(stackIDs, stack.StackID(s))
				}
				extraDomains = domains
			} else {
				// Auto-detect stacks.
				detected, detectErr := detect.Detect(targetDir)
				if detectErr != nil {
					return fmt.Errorf("detect stacks: %w", detectErr)
				}

				// Determine whether to run the wizard.
				if !nonInteractive && prompter == nil && isTerminal(cmd.InOrStdin()) {
					prompter = &wizard.HuhPrompter{}
				}
				if prompter != nil {
					var wizErr error
					choices, wizErr = prompter.Run(detected)
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

				if len(stackIDs) == 0 {
					return fmt.Errorf("no stacks detected; use --stack to specify manually")
				}
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Stacks: %v\n", stackIDs)

			// Build version overrides: initialize from wizard, then layer CLI flags on top.
			versionOverrides := make(map[string]string)
			// Wizard overrides (from interactive prompting).
			for k, v := range choices.RuntimeVersions {
				versionOverrides[k] = v
			}
			// CLI flag overrides take precedence (for scripted use).
			if len(runtimeVersions) > 0 {
				runtimeVersions = trimAndFilter(runtimeVersions)
				parsed, parseErr := parseRuntimeVersions(runtimeVersions)
				if parseErr != nil {
					return parseErr
				}
				for k, v := range parsed {
					versionOverrides[k] = v
				}
			}

			// Render all agentbox-managed files.
			files, err := renderFiles(stackIDs, extraDomains, versionOverrides)
			if err != nil {
				return err
			}

			// Append the custom stage stub to the Dockerfile.
			customStage, err := render.CustomStage()
			if err != nil {
				return err
			}
			files["Dockerfile"] = append(files["Dockerfile"], '\n')
			files["Dockerfile"] = append(files["Dockerfile"], []byte(customStage)...)

			// Write .devcontainer/ directory.
			if err := os.MkdirAll(outDir, 0o755); err != nil {
				return fmt.Errorf("create .devcontainer: %w", err)
			}

			for name, content := range files {
				path := filepath.Join(outDir, name)
				if err := os.WriteFile(path, content, 0o644); err != nil {
					return fmt.Errorf("write %s: %w", name, err)
				}
			}

			// Make shell scripts executable.
			// Intentionally coupled with cmd/update.go executable scripts list -- update both together.
			for _, name := range []string{"init-firewall.sh", "warmup-dns.sh", "sync-claude-settings.sh"} {
				path := filepath.Join(outDir, name)
				if err := os.Chmod(path, 0o755); err != nil {
					return fmt.Errorf("chmod %s: %w", name, err)
				}
			}

			// Write .agentbox.yml config file.
			agentboxCfg := config.Config{
				Version:         1,
				Stacks:          stackIDsToStrings(stackIDs),
				ExtraDomains:    extraDomains,
				GeneratedAt:     time.Now().UTC(),
				AgentboxVersion: version,
			}
			var cfgBuf bytes.Buffer
			if err := config.Write(&cfgBuf, agentboxCfg); err != nil {
				return fmt.Errorf("render %s: %w", config.Filename, err)
			}
			if err := os.WriteFile(filepath.Join(targetDir, config.Filename), cfgBuf.Bytes(), 0o644); err != nil {
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
	cmd.Flags().StringSliceVar(&runtimeVersions, "runtime-version", nil,
		"Runtime version overrides as tool=version pairs (e.g., go=1.22,node=20)")

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

// resolveDir resolves the target directory from the --dir flag value.
func resolveDir(dir string) (string, error) {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
		return wd, nil
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", dir, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("--dir %q: %w", dir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("--dir %q: not a directory", dir)
	}

	return abs, nil
}

// trimAndFilter trims whitespace and removes empty strings from a slice.
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

// validateStackIDs checks that all provided stack ID strings are valid.
func validateStackIDs(ids []string) error {
	for _, id := range ids {
		if _, ok := stack.Get(stack.StackID(id)); !ok {
			return fmt.Errorf("unknown stack %q; valid stacks: %v", id, stack.IDs())
		}
	}
	return nil
}

// stackIDsToStrings converts a slice of stack.StackID to []string.
func stackIDsToStrings(ids []stack.StackID) []string {
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = string(id)
	}
	return result
}

// renderFiles produces all agentbox-managed file content for the given
// configuration. It returns a map from filename to content. The Dockerfile
// value contains only the agentbox stage (no custom stage). The caller is
// responsible for appending the custom stage (init) or preserving the
// existing one (update).
//
// versionOverrides maps tool names to version strings (e.g., "go" -> "1.22").
// Entries that do not match any runtime in the merged config are silently
// ignored (no coupling to the registry).
func renderFiles(stackIDs []stack.StackID, extraDomains []string, versionOverrides map[string]string) (map[string][]byte, error) {
	cfg, err := render.Merge(stackIDs, extraDomains)
	if err != nil {
		return nil, fmt.Errorf("merge config: %w", err)
	}

	render.EnsureNode(&cfg)

	// Apply version overrides to runtimes.
	for i, rt := range cfg.Runtimes {
		if v, ok := versionOverrides[rt.Tool]; ok && v != "" {
			cfg.Runtimes[i].Version = v
		}
	}

	// Render all templates.
	dockerfile, err := render.Dockerfile(cfg)
	if err != nil {
		return nil, err
	}

	var devcontainerBuf bytes.Buffer
	if err := render.DevContainer(&devcontainerBuf, cfg); err != nil {
		return nil, err
	}

	fw, err := render.RenderFirewall(cfg)
	if err != nil {
		return nil, err
	}

	cl, err := render.RenderClaude(cfg)
	if err != nil {
		return nil, err
	}

	readme, err := render.README(cfg)
	if err != nil {
		return nil, err
	}

	var miseConfigBuf bytes.Buffer
	if err := render.MiseConfig(&miseConfigBuf, cfg); err != nil {
		return nil, err
	}

	return map[string][]byte{
		"Dockerfile":                []byte(dockerfile),
		"devcontainer.json":         devcontainerBuf.Bytes(),
		"init-firewall.sh":          fw.InitFirewall,
		"warmup-dns.sh":             fw.WarmupDNS,
		"dynamic-domains.conf":      fw.DynamicDomains,
		"claude-user-settings.json": cl.UserSettings,
		"sync-claude-settings.sh":   cl.SyncSettings,
		"README.md":                 []byte(readme),
		"config.toml":               miseConfigBuf.Bytes(),
	}, nil
}

// parseRuntimeVersions parses "tool=version" pairs into a map.
func parseRuntimeVersions(pairs []string) (map[string]string, error) {
	result := make(map[string]string, len(pairs))
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --runtime-version %q; expected tool=version format (e.g., go=1.22)", pair)
		}
		tool := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])
		if tool == "" || version == "" {
			return nil, fmt.Errorf("invalid --runtime-version %q; expected tool=version format (e.g., go=1.22)", pair)
		}
		result[tool] = version
	}
	return result, nil
}
