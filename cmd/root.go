package cmd

import (
	"github.com/spf13/cobra"
)

// version is set at build time via ldflags:
//
//	go build -ldflags "-X github.com/bjro/ccbox/cmd.version=1.0.0"
var version = "dev"

var rootCmd = newRootCmd()

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ccbox",
		Short:         "Generate devcontainer setups for Claude Code",
		Long:          "ccbox generates .devcontainer/ configurations for running Claude Code in sandboxed environments with full permissions and network isolation.",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.SetVersionTemplate("ccbox version {{.Version}}\n")

	cmd.AddCommand(newInitCmd())

	return cmd
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}
