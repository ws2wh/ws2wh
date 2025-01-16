package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/ws2wh/ws2wh/server"
)

// main starts the WS2WH server with configuration from flags or environment variables
// Flags:
// -b, BACKEND_URL: Required - Webhook backend URL that will receive POST requests
// -r, REPLY_PATH_PREFIX: Path prefix for backend replies (default: /reply)
// -l, WS_PORT: Address and port for WebSocket server to listen on (default: :3000)
// -p, WS_PATH: Path where WebSocket connections will be upgraded (default: /)
func main() {
	backendUrl := flag.String("b", getEnvOrDefault("BACKEND_URL", ""), "Required - Webhook backend URL (must accept POST)")
	replyPathPrefix := flag.String("r", getEnvOrDefault("REPLY_PATH_PREFIX", "/reply"), "Backend reply path prefix")
	websocketListener := flag.String("l", fmt.Sprintf(":%s", getEnvOrDefault("WS_PORT", "3000")), "Websocket frontend listener address")
	websocketPath := flag.String("p", getEnvOrDefault("WS_PATH", "/"), "Websocket upgrade path")
	logLevel := flag.String("v", getEnvOrDefault("LOG_LEVEL", "INFO"), "Log level (DEBUG,	INFO, WARN, ERROR, OFF; default: INFO)")

	flag.Parse()
	if *backendUrl == "" {
		log.Fatalf("Webhook backend URL is required")
	}
	_, e := url.ParseRequestURI(*backendUrl)
	if e != nil {
		log.Fatalf("Invalid backend URL: %s", *backendUrl)
	}

	server.CreateServer(*websocketListener, *websocketPath, *backendUrl, *replyPathPrefix, *logLevel).Start()
}

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
