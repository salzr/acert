package agent

import (
	"github.com/salzr/acert/agent"
	"github.com/spf13/cobra"
)

// TODO: create flags for ca and certificate

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use: "agent",
		Run: func(cmd *cobra.Command, args []string) {
			agent.Run(cmd.Context())
		},
	}
	cmd.AddCommand(agentInit())

	return cmd
}
