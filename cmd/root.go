package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "local-agi",
	Short: "LocalAGI - Self-hosted AI Agent platform",
	Long:  "LocalAGI is a self-hosted AI Agent platform that allows running autonomous agents with various connectors, actions, and tools.",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(agentCmd)
}
