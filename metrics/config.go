package metrics

// MetricsConfig is the configuration for the metrics server (Prometheus)
type MetricsConfig struct {
	// Enabled toggles the metrics server (default: false)
	Enabled bool
	// Port is the port for the metrics server (default: 9090)
	Port string
	// Path is the path for the metrics server (default: /metrics)
	Path string
}
