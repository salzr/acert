package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/salzr/acert/cmd/agent"
	"github.com/salzr/acert/cmd/server"
	"github.com/salzr/acert/helm"
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
	rootCmd.AddCommand(bootstrap())
	rootCmd.AddCommand(agent.Command())
	rootCmd.AddCommand(server.Command())
}

func bootstrap() *cobra.Command {
	return &cobra.Command{
		Use: "bootstrap",
		Run: func(cmd *cobra.Command, args []string) {
			logger := cmd.Context().Value("logger").(*zap.Logger)
			logger = logger.With(zap.String("service", "bootstrap"))

			ch := helm.Chart{
				Name:       "cert-manager",
				Repository: "quay.io/jetstack/charts/cert-manager",
				Version:    "v1.19.2",
				Namespace:  "cert-manager",
			}
			rel, err := helm.InstallOrUpdate(cmd.Context(), ch,
				map[string]interface{}{
					"crds": map[string]interface{}{
						"enabled": true,
					},
				})
			if err != nil {
				logger.Fatal("Failed to install chart", zap.Error(err))
			}

			logger.Info("Chart installed successfully", zap.String("release", rel.Name))
		},
	}
}
