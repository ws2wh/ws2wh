package tests

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pmartynski/ws2wh/backend"
)

type TestWebhook struct {
	echoStack *echo.Echo
	messages  chan backend.BackendMessage
}

func CreateTestWebhook() *TestWebhook {
	b := TestWebhook{
		messages:  make(chan backend.BackendMessage, 100),
		echoStack: echo.New(),
	}

	e := b.echoStack
	e.POST("/", b.handler)

	return &b
}

func (b *TestWebhook) handler(c echo.Context) error {
	p, _ := io.ReadAll(c.Request().Body)
	msg := backend.BackendMessage{
		SessionId:    c.Request().Header.Get(backend.SessionIdHeader),
		ReplyChannel: c.Request().Header.Get(backend.ReplyChannelHeader),
		Event:        toWsEvent(c.Request().Header.Get(backend.EventHeader)),
		Payload:      p,
	}

	b.messages <- msg
	c.NoContent(204)
	return nil
}

func toWsEvent(v string) backend.WsEvent {
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

func (b *TestWebhook) Start() {
	go func() { b.echoStack.Logger.Info(b.echoStack.Start(BackendHost)) }()
}

func (b *TestWebhook) Stop() {
	b.echoStack.Shutdown(context.Background())
}

func (b *TestWebhook) WaitForMessage(t *testing.T) backend.BackendMessage {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()
	select {
	case m := <-b.messages:
		return m
	case <-ctx.Done():
		t.Log("Backend didn't receive expected message on time")
		t.Fail()
		panic("unreachable")
	}
}
