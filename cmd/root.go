package cmd

import (
	"github.com/spf13/cobra"
)

// Version is set by goreleaser via ldflags
var Version = "dev"

var rootCmd = &cobra.Command{
	Use:     "tokyo",
	Short:   "Tokyo - Manage Claude Code and Codex configuration profiles",
	Long:    `Tokyo is a CLI tool for managing Claude Code and Codex configuration profiles.`,
	Version: Version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
