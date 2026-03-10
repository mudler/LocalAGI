package cmd

import (
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agents",
	Long:  "Commands for managing and running LocalAGI agents.",
}

func init() {
	agentCmd.AddCommand(agentRunCmd)
}
