package main

import (
	"flag"
	"log"
	"net/url"

	"github.com/pmartynski/ws2wh/server"
)

func main() {
	frontAddr := flag.String("f", ":3000", "Websocket frontend listener address")
	backUrl := flag.String("b", "", "Required: Webhook backend URL (must accept POST)")
	_, e := url.Parse(*backUrl)
	if e != nil {
		log.Fatalf("Invalid backend URL: %s, err: %s", *backUrl, e)
	}

	flag.Parse()

	server.Run(*frontAddr, *backUrl)
}
