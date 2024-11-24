package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/pmartynski/ws2wh/server"
)

func main() {
	backendUrl := flag.String("b", "", "Required - Webhook backend URL (must accept POST)")
	replyPathPrefix := flag.String("r", "/reply", "Backend reply path prefix")
	websocketListener := flag.String("l", ":3000", "Websocket frontend listener address")
	websocketPath := flag.String("p", "/", "Websocket upgrade path")

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
