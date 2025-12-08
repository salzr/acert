package server

import (
	"fmt"

	"github.com/spf13/cobra"
)

var authGroup = &cobra.Group{
	Title: "Group of commands for token management",
	ID:    "auth",
}

func createToken() *cobra.Command {
	return &cobra.Command{
		GroupID: authGroup.ID,
		Use:     "create-token",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("create")
		},
	}
}
