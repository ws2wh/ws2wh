package metrics

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var server *http.Server
var logger = slog.With("category", "metrics_server")

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

	logger.Info("Starting metrics server", "port", config.Port, "path", config.Path)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Metrics server error", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		logger.Info("Context cancelled, shutting down metrics server")
		stopMetricsServer()
	}()
}

func stopMetricsServer() {
	if server == nil {
		return
	}
	logger.Info("Stopping metrics server")
	server.Shutdown(context.Background())
	logger.Info("Metrics server stopped")
}
