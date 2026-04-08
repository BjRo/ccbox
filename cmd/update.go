package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bjro/agentbox/internal/config"
	"github.com/bjro/agentbox/internal/dockerfile"
	"github.com/bjro/agentbox/internal/render"
	"github.com/bjro/agentbox/internal/stack"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var stacks []string
	var domains []string
	var dir string
	var force bool

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Regenerate agentbox-managed devcontainer files",
		Long: `Regenerate the agentbox-managed portion of .devcontainer/ while preserving
user customizations in the Dockerfile custom stage and config.toml.

To change runtime versions, edit .devcontainer/config.toml directly.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Resolve target directory.
			targetDir, err := resolveDir(dir)
			if err != nil {
				return err
			}

			// Load .agentbox.yml.
			cfgPath := filepath.Join(targetDir, config.Filename)
			cfgFile, err := os.Open(cfgPath)
			if err != nil {
				return fmt.Errorf("no %s found in %s; run 'agentbox init' first", config.Filename, targetDir)
			}
			defer func() { _ = cfgFile.Close() }()

			agentboxCfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load %s: %w", config.Filename, err)
			}

			// Determine stacks.
			stacks = trimAndFilter(stacks)
			var stackIDs []stack.StackID
			if len(stacks) > 0 {
				if err := validateStackIDs(stacks); err != nil {
					return err
				}
				for _, s := range stacks {
					stackIDs = append(stackIDs, stack.StackID(s))
				}
			} else {
				for _, s := range agentboxCfg.Stacks {
					stackIDs = append(stackIDs, stack.StackID(s))
				}
			}

			// Determine extra domains.
			domains = trimAndFilter(domains)
			var extraDomains []string
			if len(domains) > 0 {
				extraDomains = domains
			} else {
				extraDomains = agentboxCfg.ExtraDomains
			}

			// Read existing Dockerfile.
			outDir := filepath.Join(targetDir, ".devcontainer")
			dfPath := filepath.Join(outDir, "Dockerfile")
			existingDF, err := os.ReadFile(dfPath)
			if err != nil {
				return fmt.Errorf("no Dockerfile found in .devcontainer/; run 'agentbox init' first")
			}

			// Split Dockerfile at custom stage boundary.
			var userPart string
			_, userPart, err = dockerfile.SplitAtCustomStage(string(existingDF))
			if err != nil {
				if !errors.Is(err, dockerfile.ErrNoCustomStage) {
					return err
				}
				if !force {
					return fmt.Errorf("dockerfile does not contain custom stage (FROM agentbox AS custom); use --force to regenerate fully")
				}
				// --force: generate fresh custom stage.
				userPart = ""
			}

			// Read existing config.toml (preserved on update).
			configTomlPath := filepath.Join(outDir, "config.toml")
			existingConfigToml, configTomlErr := os.ReadFile(configTomlPath)

			// Render fresh agentbox-managed files.
			files, err := renderFiles(stackIDs, extraDomains, nil)
			if err != nil {
				return err
			}

			// Assemble Dockerfile: fresh agentbox stage + preserved user part.
			if userPart != "" {
				files["Dockerfile"] = append(files["Dockerfile"], '\n')
				files["Dockerfile"] = append(files["Dockerfile"], []byte(userPart)...)
			} else {
				// --force or missing custom stage: generate fresh stub.
				customStage, csErr := render.CustomStage()
				if csErr != nil {
					return csErr
				}
				files["Dockerfile"] = append(files["Dockerfile"], '\n')
				files["Dockerfile"] = append(files["Dockerfile"], []byte(customStage)...)
			}

			// Preserve existing config.toml if it was found.
			if configTomlErr == nil {
				files["config.toml"] = existingConfigToml
			}

			// Write files.
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
			for _, name := range []string{"init-firewall.sh", "warmup-dns.sh", "sync-claude-settings.sh"} {
				path := filepath.Join(outDir, name)
				if err := os.Chmod(path, 0o755); err != nil {
					return fmt.Errorf("chmod %s: %w", name, err)
				}
			}

			// Write updated .agentbox.yml.
			updatedCfg := config.Config{
				Version:         1,
				Stacks:          stackIDsToStrings(stackIDs),
				ExtraDomains:    extraDomains,
				GeneratedAt:     time.Now().UTC(),
				AgentboxVersion: version,
			}
			var cfgBuf bytes.Buffer
			if err := config.Write(&cfgBuf, updatedCfg); err != nil {
				return fmt.Errorf("render %s: %w", config.Filename, err)
			}
			if err := os.WriteFile(cfgPath, cfgBuf.Bytes(), 0o644); err != nil {
				return fmt.Errorf("write %s: %w", config.Filename, err)
			}

			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Updated .devcontainer/ in %s\n", targetDir)
			return nil
		},
	}

	cmd.Flags().StringSliceVar(&stacks, "stack", nil, "Override stacks (persists to .agentbox.yml). Auto-detects if omitted.")
	cmd.Flags().StringSliceVar(&domains, "extra-domains", nil, "Override extra domains (persists to .agentbox.yml)")
	cmd.Flags().StringVar(&dir, "dir", "", "Target directory (default: current directory)")
	cmd.Flags().BoolVar(&force, "force", false, "Force full regeneration even if custom stage is missing")

	return cmd
}
