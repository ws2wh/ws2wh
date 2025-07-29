package metrics

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	server   *http.Server
	serverMu sync.Mutex
)

func StartMetricsServer(ctx context.Context, config *MetricsConfig) {
	if config == nil || !config.Enabled {
		return
	}

	serverMu.Lock()
	defer serverMu.Unlock()

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
	serverMu.Lock()
	defer serverMu.Unlock()
	if server == nil {
		return
	}
	slog.Info("Stopping metrics server")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Metrics server shutdown error", "error", err)
		server.Close()
	}
	slog.Info("Metrics server stopped")
}
