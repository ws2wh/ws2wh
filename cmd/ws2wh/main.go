package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/ws2wh/ws2wh/cmd/logger"
	"github.com/ws2wh/ws2wh/cmd/ws2wh/flags"
	"github.com/ws2wh/ws2wh/metrics"
	"github.com/ws2wh/ws2wh/server"
)

func main() {
	// TODO: need to create integration/smoke tests for all listeners loaded from config
	config := flags.LoadConfig()
	logger.InitLogger(config)

	ctx, cancel := context.WithCancel(context.Background())

	metrics.StartMetricsServer(ctx, config.MetricsConfig)
	server.CreateServerWithConfig(config).Start(ctx)

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, os.Interrupt, os.Kill)
	<-sigs

	cancel()
}
