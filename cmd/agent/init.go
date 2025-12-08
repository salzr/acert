package agent

import (
	"fmt"

	"github.com/spf13/cobra"
)

func agentInit() *cobra.Command {
	return &cobra.Command{
		Use: "init",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("init")
		},
	}
}
