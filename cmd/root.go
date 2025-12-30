package cmd

import (
	"context"
	"fmt"

	"github.com/salzr/acert/cmd/bootstrap"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/salzr/acert/cmd/agent"
	"github.com/salzr/acert/cmd/server"
)

var rootCmd = &cobra.Command{
	Use:   "acert",
	Short: "acert is a tool for managing certificates",
	Long:  `Certificate management toolbox with a server and agent component for managing certificates`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("acert")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "logger", logger)

	rootCmd.SetContext(ctx)
	rootCmd.AddCommand(bootstrap.Command())
	rootCmd.AddCommand(agent.Command())
	rootCmd.AddCommand(server.Command())
}
