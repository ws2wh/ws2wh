package metrics

import (
	"context"
	"net/http"

	"github.com/labstack/gommon/log"

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

	log.Infoj(map[string]interface{}{
		"message": "Starting metrics server",
		"port":    config.Port,
		"path":    config.Path,
	})

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorj(map[string]interface{}{
				"message": "Metrics server error",
				"error":   err,
			})
		}
	}()

	go func() {
		<-ctx.Done()
		log.Infoj(map[string]interface{}{
			"message": "Context cancelled, shutting down metrics server",
		})
		stopMetricsServer()
	}()
}

func stopMetricsServer() {
	if server == nil {
		return
	}
	log.Infoj(map[string]interface{}{
		"message": "Stopping metrics server",
	})
	server.Shutdown(context.Background())
	log.Infoj(map[string]interface{}{
		"message": "Metrics server stopped",
	})
}
