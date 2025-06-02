package tests

import (
	"context"

	"github.com/labstack/gommon/log"
	"github.com/ws2wh/ws2wh/metrics"
	"github.com/ws2wh/ws2wh/server"
)

const WsHost = ":3000"
const WsUrl = "ws://localhost:3000"
const BackendHost = ":5000"
const BackendUrl = "http://localhost:5000"

func CreateTestWs() TestWsServer {
	ctx, cancel := context.WithCancel(context.Background())
	return TestWsServer{

		server: server.CreateServerWithConfig(&server.Config{
			BackendUrl:        BackendUrl,
			WebSocketListener: WsHost,
			WebSocketPath:     "/",
			ReplyChannelConfig: &server.ReplyChannelConfig{
				PathPrefix: "/reply",
				Hostname:   "localhost",
				Scheme:     "http",
				Port:       "3000",
			},
			LogLevel: log.DEBUG,
			Hostname: "localhost",
			MetricsConfig: &metrics.MetricsConfig{
				Enabled: false,
			},
			TlsConfig: &server.TlsConfig{
				Enabled: false,
			},
		}),

		context: ctx,
		stop:    cancel,
	}
}

type TestWsServer struct {
	server    *server.Server
	IsRunning bool
	context   context.Context
	stop      context.CancelFunc
}

func (s *TestWsServer) Start() {
	go s.server.Start(s.context)
	s.IsRunning = true
}

func (s *TestWsServer) Stop() {
	s.stop()
	s.IsRunning = false
}
