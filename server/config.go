package server

import (
	"github.com/labstack/gommon/log"
	"github.com/ws2wh/ws2wh/metrics"
)

// Config holds the server configuration parameters
type Config struct {
	// BackendUrl is the webhook backend URL that will receive POST requests
	BackendUrl string
	// ReplyPathPrefix is the path prefix for backend replies (default: /reply)
	ReplyPathPrefix string
	// ReplyUrl is the URL for backend replies (default: http://localhost:3000/reply)
	ReplyUrl string
	// WebSocketListener is the address and port for WebSocket server to listen on (default: :3000)
	WebSocketListener string
	// WebSocketPath is the path where WebSocket connections will be upgraded (default: /)
	WebSocketPath string
	// LogLevel sets the logging level (DEBUG, INFO, WARN, ERROR, OFF; default: INFO)
	LogLevel log.Lvl
	// Hostname is used in the reply channel URL (default: localhost)
	Hostname string
	// MetricsConfig holds the metrics configuration parameters
	MetricsConfig *metrics.MetricsConfig
	// TlsConfig holds the TLS configuration parameters
	TlsConfig *TlsConfig
}

type TlsConfig struct {
	// Enabled toggles TLS configuration (default: false)
	Enabled bool
	// TlsCertPath is the path to the TLS certificate file in PEM format (optional)
	TlsCertPath string
	// TlsKeyPath is the path to the TLS private key file in PEM format (optional)
	TlsKeyPath string
}
