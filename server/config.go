package server

import (
	"fmt"

	"github.com/labstack/gommon/log"
	"github.com/ws2wh/ws2wh/metrics"
)

// Config holds the server configuration parameters
type Config struct {
	// BackendUrl is the webhook backend URL that will receive POST requests
	BackendUrl string
	// ReplyChannelConfig holds the reply channel configuration parameters
	ReplyChannelConfig *ReplyChannelConfig
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

// ReplyChannelConfig holds the reply channel configuration parameters
type ReplyChannelConfig struct {
	// PathPrefix is the path prefix for the reply channel (default: /reply)
	PathPrefix string
	// Hostname is the hostname for the reply channel (default: localhost)
	Hostname string
	// Scheme is the scheme for the reply channel (default: http)
	Scheme string
	// Port is the port for the reply channel (default: 3000)
	Port string
}

func (c *ReplyChannelConfig) GetReplyUrl() string {
	return fmt.Sprintf("%s://%s:%s%s", c.Scheme, c.Hostname, c.Port, c.PathPrefix)
}
