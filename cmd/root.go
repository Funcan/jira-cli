package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "jira [cmd]",
	Short:        "A CLI for manipulating JIRA tickekts",
	Long:         `A CLI for manipulating JIRA tickets - list, create, update, close tickets`,
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
