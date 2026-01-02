package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
			h := api.NewServer()

			srv := &http.Server{
				Addr:              addr,
				Handler:           h,
				ReadHeaderTimeout: 5 * time.Second,
				ReadTimeout:       15 * time.Second,
				WriteTimeout:      30 * time.Second,
				IdleTimeout:       60 * time.Second,
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			errCh := make(chan error, 1)
			go func() {
				errCh <- srv.ListenAndServe()
			}()

			fmt.Fprintf(cmd.OutOrStdout(), "Starting server on %s\n", addr)

			select {
			case <-ctx.Done():
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				_ = srv.Shutdown(shutdownCtx)
				return nil
			case err := <-errCh:
				if err == nil || errors.Is(err, http.ErrServerClosed) {
					return nil
				}
				return err
			}
		},
	}

	cmd.Flags().StringVarP(&addr, "addr", "a", ":8080", "Address to listen on")

	return cmd
}
