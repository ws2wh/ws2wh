package tests

import (
	"github.com/labstack/gommon/log"
	"github.com/ws2wh/ws2wh/metrics"
	"github.com/ws2wh/ws2wh/server"
)

const WsHost = ":3000"
const WsUrl = "ws://localhost:3000"
const BackendHost = ":5000"
const BackendUrl = "http://localhost:5000"

func CreateTestWs() TestWsServer {
	return TestWsServer{

		server: *server.CreateServerWithConfig(&server.Config{
			BackendUrl:        BackendUrl,
			WebSocketListener: WsHost,
			WebSocketPath:     "/",
			ReplyPathPrefix:   "/reply",
			ReplyUrl:          "http://localhost:3000/reply",
			LogLevel:          log.DEBUG,
			Hostname:          "localhost",
			MetricsConfig: &metrics.MetricsConfig{
				Enabled: false,
			},
			TlsConfig: &server.TlsConfig{
				Enabled: false,
			},
		}),
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
