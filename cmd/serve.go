package cmd

import (
	"fmt"
	"net/http"

	"tokyo/api"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newServeCommand())
}

func newServeCommand() *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the HTTP API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			server := api.NewServer()
			fmt.Fprintf(cmd.OutOrStdout(), "Starting server on %s\n", addr)
			return http.ListenAndServe(addr, server)
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "a", ":8080", "Address to listen on")

	return cmd
}
