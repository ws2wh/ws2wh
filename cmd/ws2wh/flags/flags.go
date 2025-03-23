package flags

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/labstack/gommon/log"
	"github.com/ws2wh/ws2wh/metrics"
	"github.com/ws2wh/ws2wh/server"
)

func LoadConfig() *server.Config {

	backendUrl := flag.String("b", getEnvOrDefault("BACKEND_URL", ""), "Required - Webhook backend URL (must accept POST)")
	replyPathPrefix := flag.String("r", getEnvOrDefault("REPLY_PATH_PREFIX", "/reply"), "Backend reply path prefix")
	websocketListener := flag.String("l", fmt.Sprintf(":%s", getEnvOrDefault("WS_PORT", "3000")), "Websocket frontend listener address")
	websocketPath := flag.String("p", getEnvOrDefault("WS_PATH", "/"), "Websocket upgrade path")
	logLevel := flag.String("v", getEnvOrDefault("LOG_LEVEL", "INFO"), "Log level (DEBUG,	INFO, WARN, ERROR, OFF; default: INFO)")
	hostname := flag.String("h", getEnvOrDefault("REPLY_HOSTNAME", getEnvOrDefault("HOSTNAME", "localhost")), "Hostname to use in reply channel")
	enableMetrics := flag.String("metrics-enabled", getEnvOrDefault("METRICS_ENABLED", "false"), "Enable Prometheus metrics")
	metricsPort := flag.String("metrics-port", getEnvOrDefault("METRICS_PORT", "9090"), "Prometheus metrics port")
	metricsPath := flag.String("metrics-path", getEnvOrDefault("METRICS_PATH", "/metrics"), "Prometheus metrics path")
	tlsEnabled := flag.String("tls-enabled", getEnvOrDefault("TLS_ENABLED", "false"), "Enable TLS")
	tlsCertPath := flag.String("tls-cert-path", getEnvOrDefault("TLS_CERT_PATH", ""), "(Optional) TLS certificate path (PEM format). Required if TLS key path set.")
	tlsKeyPath := flag.String("tls-key-path", getEnvOrDefault("TLS_KEY_PATH", ""), "(Optional) TLS key path (PEM format). Required if TLS certificate path set.")

	flag.Parse()

	if *backendUrl == "" {
		log.Fatalf("Webhook backend URL is required")
	}
	_, e := url.ParseRequestURI(*backendUrl)
	if e != nil {
		log.Fatalf("Invalid backend URL: %s", *backendUrl)
	}

	if *tlsCertPath != "" && *tlsKeyPath == "" {
		log.Fatalf("TLS certificate path set but TLS key path not set")
	}

	if *tlsCertPath == "" && *tlsKeyPath != "" {
		log.Fatalf("TLS key path set but TLS certificate path not set")
	}

	var replyScheme string
	if *tlsCertPath != "" && *tlsKeyPath != "" {
		replyScheme = "https"
	} else {
		replyScheme = "http"
	}
	replyUrl := fmt.Sprintf("%s://%s%s%s", replyScheme, *hostname, *websocketListener, *replyPathPrefix)

	if *enableMetrics == "true" {
		metrics.StartMetricsServer(*metricsPort, *metricsPath)
	}

	return &server.Config{
		BackendUrl: *backendUrl,
		// TODO: rearrange how reply channel is handled:
		// - set ReplyPathPrefix
		// - set ReplyChannelHostname
		// - set ReplyChannelScheme
		// - consider separate port for reply channel, for future use with separate listeners
		ReplyPathPrefix:   *replyPathPrefix,
		ReplyUrl:          replyUrl,
		WebSocketListener: *websocketListener,
		WebSocketPath:     *websocketPath,
		LogLevel:          parse(*logLevel),
		Hostname:          *hostname,

		// TODO: move elsewhere - not required for server
		MetricsConfig: &metrics.MetricsConfig{
			Enabled: *enableMetrics == "true",
			Port:    *metricsPort,
			Path:    *metricsPath,
		},
		TlsConfig: &server.TlsConfig{
			Enabled:     *tlsEnabled == "true",
			TlsCertPath: *tlsCertPath,
			TlsKeyPath:  *tlsKeyPath,
		},
	}
}

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func parse(logLevel string) log.Lvl {
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		return log.DEBUG
	case "INFO":
		return log.INFO
	case "WARN":
		return log.WARN
	case "ERROR":
		return log.ERROR
	case "OFF":
		return log.OFF
	}

	log.Warnf("Unknown log level: %s, using INFO instead", logLevel)
	return log.INFO
}
