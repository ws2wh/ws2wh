package main

import (
	"github.com/ws2wh/ws2wh/cmd/ws2wh/flags"
	"github.com/ws2wh/ws2wh/server"
)

// main starts the WS2WH server with configuration from flags or environment variables
// Flags:
// -b, BACKEND_URL: Required - Webhook backend URL that will receive POST requests
// -r, REPLY_PATH_PREFIX: Path prefix for backend replies (default: /reply)
// -l, WS_PORT: Address and port for WebSocket server to listen on (default: :3000)
// -p, WS_PATH: Path where WebSocket connections will be upgraded (default: /)
// -v, LOG_LEVEL: Log level (DEBUG, INFO, WARN, ERROR, OFF; default: INFO)
// -h, REPLY_HOSTNAME: Hostname to use in reply channel (default: localhost)
// -metrics-port, METRICS_PORT: Port for metrics endpoint (default: 9090)
// -metrics-path, METRICS_PATH: Path for metrics endpoint (default: /metrics)
// -metrics-enabled, METRICS_ENABLED: Enable metrics collection (default: false)
func main() {
	// backendUrl := flag.String("b", getEnvOrDefault("BACKEND_URL", ""), "Required - Webhook backend URL (must accept POST)")
	// replyPathPrefix := flag.String("r", getEnvOrDefault("REPLY_PATH_PREFIX", "/reply"), "Backend reply path prefix")
	// websocketListener := flag.String("l", fmt.Sprintf(":%s", getEnvOrDefault("WS_PORT", "3000")), "Websocket frontend listener address")
	// websocketPath := flag.String("p", getEnvOrDefault("WS_PATH", "/"), "Websocket upgrade path")
	// logLevel := flag.String("v", getEnvOrDefault("LOG_LEVEL", "INFO"), "Log level (DEBUG,	INFO, WARN, ERROR, OFF; default: INFO)")
	// hostname := flag.String("h", getEnvOrDefault("REPLY_HOSTNAME", getEnvOrDefault("HOSTNAME", "localhost")), "Hostname to use in reply channel")
	// metricsPort := flag.String("metrics-port", getEnvOrDefault("METRICS_PORT", "9090"), "Prometheus metrics port")
	// metricsPath := flag.String("metrics-path", getEnvOrDefault("METRICS_PATH", "/metrics"), "Prometheus metrics path")
	// enableMetrics := flag.String("metrics-enabled", getEnvOrDefault("METRICS_ENABLED", "false"), "Enable Prometheus metrics")
	// tlsCertPath := flag.String("tls-cert-path", getEnvOrDefault("TLS_CERT_PATH", ""), "(Optional) TLS certificate path (PEM format). Required if TLS key path set.")
	// tlsKeyPath := flag.String("tls-key-path", getEnvOrDefault("TLS_KEY_PATH", ""), "(Optional) TLS key path (PEM format). Required if TLS certificate path set.")

	// flag.Parse()
	// if *backendUrl == "" {
	// 	log.Fatalf("Webhook backend URL is required")
	// }
	// _, e := url.ParseRequestURI(*backendUrl)
	// if e != nil {
	// 	log.Fatalf("Invalid backend URL: %s", *backendUrl)
	// }

	// if *tlsCertPath != "" && *tlsKeyPath == "" {
	// 	log.Fatalf("TLS certificate path set but TLS key path not set")
	// }

	// if *tlsCertPath == "" && *tlsKeyPath != "" {
	// 	log.Fatalf("TLS key path set but TLS certificate path not set")
	// }

	// var replyScheme string
	// if *tlsCertPath != "" && *tlsKeyPath != "" {
	// 	replyScheme = "https"
	// } else {
	// 	replyScheme = "http"
	// }
	// replyUrl := fmt.Sprintf("%s://%s%s%s", replyScheme, *hostname, *websocketListener, *replyPathPrefix)

	// if *enableMetrics == "true" {
	// 	metrics.StartMetricsServer(*metricsPort, *metricsPath)
	// }

	config := flags.LoadConfig()
	server.CreateServerWithConfig(config).Start()
}
