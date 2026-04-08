package cmd

import (
	"github.com/bjro/agentbox/internal/wizard"
	"github.com/spf13/cobra"
)

// version is set at build time via ldflags:
//
//	go build -ldflags "-X github.com/bjro/agentbox/cmd.version=1.0.0"
var version = "dev"

var rootCmd = newRootCmd(nil)

// newRootCmd builds the full command tree. The prompter parameter is
// threaded to newInitCmd for the interactive wizard. Production code
// passes nil (HuhPrompter is used when stdin is a terminal); tests
// pass a fake.
func newRootCmd(prompter wizard.Prompter) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "agentbox",
		Short:         "Generate devcontainer setups for AI coding agents",
		Long:          "agentbox generates .devcontainer/ configurations for running AI coding agents in sandboxed environments with full permissions and network isolation.",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.SetVersionTemplate("agentbox version {{.Version}}\n")

	cmd.AddCommand(newInitCmd(prompter))
	cmd.AddCommand(newUpdateCmd())

	return cmd
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}
