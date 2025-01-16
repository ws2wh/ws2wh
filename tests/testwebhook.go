package tests

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/ws2wh/ws2wh/backend"
)

type TestWebhook struct {
	echoStack *echo.Echo
	messages  chan backend.BackendMessage
	responses [][]byte
}

func CreateTestWebhook() *TestWebhook {
	b := TestWebhook{
		messages:  make(chan backend.BackendMessage, 100),
		echoStack: echo.New(),
		responses: make([][]byte, 0),
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
		Event:        backend.ParseWsEvent(c.Request().Header.Get(backend.EventHeader)),
		Payload:      p,
	}

	b.messages <- msg
	if len(b.responses) == 0 {
		c.NoContent(204)
	} else {
		resp := b.responses[0]
		b.responses = b.responses[1:]
		c.Blob(200, "text/plain", resp)
	}

	return nil
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
		t.Errorf("test backend should receive websocket message on time")
		panic("unreachable")
	}
}
