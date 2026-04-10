package graceful

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

// Serve a http handler with graceful shutdown of connections on
// SIGINT and SIGTERM.
func Serve(server *http.Server, shutdownDuration time.Duration) {
	go listenAndServe(server)
	log.Info().Msg("Started HTTP server")
	serveGracefully(server, shutdownDuration)
}

func listenAndServe(server *http.Server) {
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Err(err).Msg("Failed to serve")
	}
}

// ServeTLS serves an HTTPS handler with graceful shutdown of connections on
// SIGINT and SIGTERM. The server's TLSConfig must have at least one certificate
// configured.
func ServeTLS(server *http.Server, shutdownDuration time.Duration) {
	if server.TLSConfig == nil || len(server.TLSConfig.Certificates) == 0 {
		panic("ServeTLS requires TLSConfig with at least one certificate")
	}
	go listenAndServeTLS(server)
	log.Info().Msg("Started HTTPS server")
	serveGracefully(server, shutdownDuration)
}

func listenAndServeTLS(server *http.Server) {
	// cert/key are in server.TLSConfig, so pass empty strings
	if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
		log.Err(err).Msg("Failed to serve")
	}
}

func serveGracefully(server *http.Server, shutdownDuration time.Duration) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signalChan)
	<-signalChan
	log.Info().Msg("Received termination signal")

	ctx, cancel := context.WithTimeout(context.Background(), shutdownDuration)
	defer cancel()
	log.Info().Msg("Closing server")
	if err := server.Shutdown(ctx); err != nil {
		log.Err(err).Msg("Server failed to shutdown")
	}
	log.Info().Msg("Server exited")
}
