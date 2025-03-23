package tests

import (
	"github.com/labstack/gommon/log"
	"github.com/ws2wh/ws2wh/server"
)

const WsHost = ":3000"
const WsUrl = "ws://localhost:3000"
const BackendHost = ":5000"
const BackendUrl = "http://localhost:5000"

func CreateTestWs() TestWsServer {
	return TestWsServer{
		server: *server.CreateServer(
			WsHost,
			"/", BackendUrl,
			"/reply",
			log.DEBUG,
			"",
			"",
			"http://localhost:3000/reply",
		),
	}
}

type TestWsServer struct {
	server    server.Server
	IsRunning bool
}

func (s *TestWsServer) Start() {
	go s.server.Start()
	s.IsRunning = true
}

func (s *TestWsServer) Stop() {
	s.server.Stop()
	s.IsRunning = false
}
