package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/pmartynski/ws2wh/server"
)

func main() {
	backendUrl := flag.String("b", getEnvOrDefault("BACKEND_URL", ""), "Required - Webhook backend URL (must accept POST)")
	replyPathPrefix := flag.String("r", getEnvOrDefault("REPLY_PATH_PREFIX", "/reply"), "Backend reply path prefix")
	websocketListener := flag.String("l", fmt.Sprintf(":%s", getEnvOrDefault("WS_PORT", "3000")), "Websocket frontend listener address")
	websocketPath := flag.String("p", getEnvOrDefault("WS_PATH", "/"), "Websocket upgrade path")

	flag.Parse()
	if *backendUrl == "" {
		flag.CommandLine.Usage()
		fmt.Printf("Webhook backend URL is required\n")
		os.Exit(1)
	}
	_, e := url.Parse(*backendUrl)
	if e != nil {
		flag.CommandLine.Usage()
		fmt.Printf("Invalid backend URL: %s, err: %s\n", *backendUrl, e)
		os.Exit(1)
	}

	server.CreateServer(*websocketListener, *websocketPath, *backendUrl, *replyPathPrefix).Start()
}

func getEnvOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
