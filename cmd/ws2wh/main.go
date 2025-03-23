package main

import (
	"github.com/ws2wh/ws2wh/cmd/ws2wh/flags"
	"github.com/ws2wh/ws2wh/metrics"
	"github.com/ws2wh/ws2wh/server"
)

func main() {
	// TODO: all listeners should be started here with a context cancelled with a signal
	// TODO: need to create integration/smoke tests for all listeners loaded from config
	config := flags.LoadConfig()
	if config.MetricsConfig != nil && config.MetricsConfig.Enabled {
		metrics.StartMetricsServer(config.MetricsConfig.Port, config.MetricsConfig.Path)
	}

	server.CreateServerWithConfig(config).Start()
}
