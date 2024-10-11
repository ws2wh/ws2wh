package main

import "github.com/pmartynski/ws2wh/server"

func main() {
	(&server.Server{
		WsPort:     3000,
		WhEndpoint: "http://localhost:3001",
	}).Serve()
}
