package flags

import (
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/ws2wh/ws2wh/http-middleware/jwt"
	"github.com/ws2wh/ws2wh/metrics"
	"github.com/ws2wh/ws2wh/server"
)

func LoadConfig() *server.Config {

	backendUrl := flag.String("b", getEnvOrDefault("BACKEND_URL", ""), "Required - Webhook backend URL (must accept POST)")
	replyPathPrefix := flag.String("r", getEnvOrDefault("REPLY_PATH_PREFIX", "/reply"), "Backend reply path prefix")
	websocketListener := flag.String("l", fmt.Sprintf(":%s", getEnvOrDefault("WS_PORT", "3000")), "Websocket frontend listener address")
	websocketPath := flag.String("p", getEnvOrDefault("WS_PATH", "/"), "Websocket upgrade path")
	logLevel := flag.String("v", getEnvOrDefault("LOG_LEVEL", "INFO"), "Log level (DEBUG,	INFO, WARN, ERROR; default: INFO)")
	hostname := flag.String("h", getEnvOrDefault("REPLY_HOSTNAME", getEnvOrDefault("HOSTNAME", "localhost")), "Hostname to use in reply channel")
	enableMetrics := flag.String("metrics-enabled", getEnvOrDefault("METRICS_ENABLED", "false"), "Enable Prometheus metrics")
	metricsPort := flag.String("metrics-port", getEnvOrDefault("METRICS_PORT", "9090"), "Prometheus metrics port")
	metricsPath := flag.String("metrics-path", getEnvOrDefault("METRICS_PATH", "/metrics"), "Prometheus metrics path")
	tlsEnabled := flag.String("tls-enabled", getEnvOrDefault("TLS_ENABLED", "false"), "Enable TLS")
	tlsCertPath := flag.String("tls-cert-path", getEnvOrDefault("TLS_CERT_PATH", ""), "(Optional) TLS certificate path (PEM format). Required if TLS key path set.")
	tlsKeyPath := flag.String("tls-key-path", getEnvOrDefault("TLS_KEY_PATH", ""), "(Optional) TLS key path (PEM format). Required if TLS certificate path set.")
	jwtEnable := flag.String("jwt-enabled", getEnvOrDefault("JWT_ENABLED", "false"), "Enable JWT authentication")
	jwtIssuer := flag.String("jwt-issuer", getEnvOrDefault("JWT_ISSUER", ""), "JWT issuer")
	jwtAudience := flag.String("jwt-audience", getEnvOrDefault("JWT_AUDIENCE", ""), "JWT audience")
	jwtSecretType := flag.String("jwt-secret-type", getEnvOrDefault("JWT_SECRET_TYPE", "jwks-url"), "JWT secret type (jwks-file, jwks-url, openid)")
	jwtSecretPath := flag.String("jwt-secret-path", getEnvOrDefault("JWT_SECRET_PATH", ""), "Path to JWT secret (file path or URL depending on secret type)")
	jwtQueryParam := flag.String("jwt-query-param", getEnvOrDefault("JWT_QUERY_PARAM", "token"), "Query parameter name for JWT token")

	flag.Parse()

	if *backendUrl == "" {
		slog.Error("Webhook backend URL is required")
		os.Exit(1)
	}
	_, e := url.ParseRequestURI(*backendUrl)
	if e != nil {
		slog.Error("Invalid backend URL", "error", e)
		os.Exit(1)
	}

	if *tlsCertPath != "" && *tlsKeyPath == "" {
		slog.Error("TLS certificate path set but TLS key path not set")
		os.Exit(1)
	}

	if *tlsCertPath == "" && *tlsKeyPath != "" {
		slog.Error("TLS key path set but TLS certificate path not set")
		os.Exit(1)
	}

	var replyScheme string
	if *tlsCertPath != "" && *tlsKeyPath != "" {
		replyScheme = "https"
	} else {
		replyScheme = "http"
	}

	return &server.Config{
		BackendUrl: *backendUrl,
		ReplyChannelConfig: &server.ReplyChannelConfig{
			PathPrefix: *replyPathPrefix,
			Hostname:   *hostname,
			Scheme:     replyScheme,
			Port: func() string {
				if strings.HasPrefix(*websocketListener, ":") {
					return (*websocketListener)[1:]
				}
				if lastColon := strings.LastIndex(*websocketListener, ":"); lastColon != -1 {
					return (*websocketListener)[lastColon+1:]
				}
				return "3000" // fallback
			}(),
		},
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
		JwtConfig: &jwt.JwtConfig{
			Enabled:      *jwtEnable == "true",
			QueryParam:   *jwtQueryParam,
			SecretSource: createSecretProvider(*jwtSecretType, *jwtSecretPath),
			Issuer:       *jwtIssuer,
			Audience:     *jwtAudience,
		},
	}
}

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func parse(logLevel string) slog.Level {
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	}

	slog.Warn("Unknown log level, using INFO instead", "logLevel", logLevel)
	return slog.LevelInfo
}

func createSecretProvider(secretType, secretPath string) jwt.KeyProvider {
	switch secretType {
	case "jwks-file":
		return &jwt.JWKSFileProvider{
			FilePath: secretPath,
		}
	case "jwks-url":
		return &jwt.JWKSURLProvider{
			URL: secretPath,
		}
	case "openid":
		return &jwt.OpenIDConfigProvider{
			Issuer: secretPath,
		}
	default:
		slog.Error("Unknown JWT secret type", "type", secretType)
		os.Exit(1)
		return nil
	}
}
