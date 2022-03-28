package cmd

import (
	"context"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const name = "whimsy"

func init() {
	root.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "run the server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := newContext()
		defer cancel()

		router, cleanup, err := buildRouter(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create router")
		}

		defer cleanup()

		addr := viper.GetString("http.address")
		httpServer := &http.Server{
			ReadTimeout:    30 * time.Second,
			WriteTimeout:   60 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler:        router,
			Addr:           addr,
		}

		// tls cert handled here, if any
		tlsCert := viper.GetString("http.tls.cert")
		tlsKey := viper.GetString("http.tls.key")

		// Launch the app, visit localhost:5000/
		go func() {
			log.Info().Str("addr", addr).Msg("Server running")
			var err error
			if len(tlsCert) > 0 && len(tlsKey) > 0 {
				err = httpServer.ListenAndServeTLS(tlsCert, tlsKey)
			} else {
				err = httpServer.ListenAndServe()
			}
			if err != nil && err != http.ErrServerClosed {
				log.Err(err).Msg("Server failed")
			}
			log.Info().Msg("Server stopped serving")
			cancel()
		}()

		<-ctx.Done()
		if err := httpServer.Shutdown(context.Background()); err != nil {
			log.Err(err).Msg("Server shutdown failed")
		}
	},
}
