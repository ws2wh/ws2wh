package metrics

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var server *http.Server

func StartMetricsServer(ctx context.Context, config *MetricsConfig) {
	if config == nil || !config.Enabled {
		return
	}

	mux := http.NewServeMux()
	mux.Handle(config.Path, promhttp.Handler())
	server = &http.Server{
		Addr:    ":" + config.Port,
		Handler: mux,
	}

	slog.Info("Starting metrics server", "port", config.Port, "path", config.Path)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Metrics server error", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		slog.Info("Context cancelled, shutting down metrics server")
		stopMetricsServer()
	}()
}

func stopMetricsServer() {
	if server == nil {
		return
	}
	slog.Info("Stopping metrics server")
	server.Shutdown(context.Background())
	slog.Info("Metrics server stopped")
}
