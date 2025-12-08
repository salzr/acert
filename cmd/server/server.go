package server

import (
	"github.com/salzr/acert/server"
	"github.com/spf13/cobra"
)

// TODO: create flags for ca and certificate

func Command() *cobra.Command {
	opts := server.DefaultOptions()
	cmd := &cobra.Command{
		Use: "server",
		Run: func(cmd *cobra.Command, args []string) {
			server.Run(cmd.Context(), opts)
		},
	}
	cmd.PersistentFlags().IntVar(&opts.GRPCPort, "grpc-port", opts.GRPCPort, "grpc port to bind to")
	cmd.PersistentFlags().IntVar(&opts.Port, "port", opts.Port, "port to bind to")

	cmd.AddGroup(authGroup)
	cmd.AddCommand(createToken())
	return cmd
}
