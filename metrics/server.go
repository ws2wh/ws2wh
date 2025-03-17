package metrics

import (
	"context"
	"net/http"

	"github.com/labstack/gommon/log"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var server *http.Server

func StartMetricsServer(port string, path string) {
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.Handler())
	server = &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}
	go func() {
		log.Infoj(map[string]interface{}{
			"message": "Starting metrics server",
			"port":    port,
			"path":    path,
		})
		server.ListenAndServe()
	}()
}

func StopMetricsServer(ctx context.Context) {
	if server == nil {
		return
	}
	log.Infoj(map[string]interface{}{
		"message": "Stopping metrics server",
	})
	server.Shutdown(ctx)
	log.Infoj(map[string]interface{}{
		"message": "Metrics server stopped",
	})
}
