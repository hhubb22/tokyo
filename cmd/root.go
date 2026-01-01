package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tokyo",
	Short: "Tokyo - A CLI tool",
	Long: `Tokyo is a CLI application built with Cobra.
This application provides various commands to help you with your tasks.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show help if no subcommand is provided
		cmd.Help()
	},
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add global flags here
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.tokyo.yaml)")
}
