package tests

import (
	"context"
	"io"

	"github.com/labstack/echo/v4"
	"github.com/pmartynski/ws2wh/backend"
	"github.com/pmartynski/ws2wh/server"
	"golang.org/x/net/websocket"
)

const WsHost = ":3000"
const WsUrl = "ws://localhost:3000"
const BackendHost = ":5000"
const BackendUrl = "http://localhost:5000"
const OriginUrl = "http://localhost"

func CreateWs() WsServer {
	return WsServer{
		server: *server.CreateServer(WsHost, BackendUrl),
	}
}

type WsServer struct {
	server    server.Server
	IsRunning bool
}

func (s *WsServer) Start() {
	go s.server.Start()
	s.IsRunning = true
}

func (s *WsServer) Stop() {
	s.server.Stop()
	s.IsRunning = false
}

func WaitForMessage(ws *websocket.Conn, out chan []byte) {
	var incomingMsg []byte
	websocket.Message.Receive(ws, &incomingMsg)
	out <- incomingMsg
}

type TestBackend struct {
	echoStack *echo.Echo
	messages  chan backend.WsMessage
}

func CreateBackend() *TestBackend {
	b := TestBackend{
		messages:  make(chan backend.WsMessage, 100),
		echoStack: echo.New(),
	}

	e := b.echoStack
	e.POST("/", b.handler)

	return &b
}

func (b *TestBackend) handler(c echo.Context) error {
	p, _ := io.ReadAll(c.Request().Body)
	msg := backend.WsMessage{
		SessionId:    c.Request().Header.Get(backend.SessionIdHeader),
		ReplyChannel: c.Request().Header.Get(backend.ReplyChannelHeader),
		Event:        ToWsEvent(c.Request().Header.Get(backend.EventHeader)),
		Payload:      p,
	}

	b.messages <- msg
	c.NoContent(204)
	return nil
}

func ToWsEvent(v string) backend.WsEvent {
	switch v {
	case "ClientConnected":
		return backend.ClientConnected
	case "MessageReceived":
		return backend.MessageReceived
	case "ClientDisconnected":
		return backend.ClientDisconnected
	default:
		return backend.WsEvent(0)
	}
}

func (b *TestBackend) Start() {
	go func() { b.echoStack.Logger.Info(b.echoStack.Start(BackendHost)) }()
}

func (b *TestBackend) Stop() {
	b.echoStack.Shutdown(context.Background())
}

func (b *TestBackend) WaitForMessage() backend.WsMessage {
	return <-b.messages
}
