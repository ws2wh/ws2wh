package tests

import (
	"github.com/pmartynski/ws2wh/server"
)

const WsHost = ":3000"
const WsUrl = "ws://localhost:3000"
const BackendHost = ":5000"
const BackendUrl = "http://localhost:5000"
const OriginUrl = "http://localhost"

func CreateTestWs() TestWsServer {
	return TestWsServer{
		server: *server.CreateServer(WsHost, "/", BackendUrl, "/reply"),
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
