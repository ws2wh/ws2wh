package tests

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/ws2wh/ws2wh/backend"
)

type TestWebhook struct {
	httpHandler http.Handler
	messages    chan backend.BackendMessage
	responses   [][]byte
	server      *http.Server
}

func CreateTestWebhook() *TestWebhook {
	httpHandler := mux.NewRouter()
	b := TestWebhook{
		messages:    make(chan backend.BackendMessage, 100),
		httpHandler: httpHandler,
		responses:   make([][]byte, 0),
		server:      nil,
	}

	httpHandler.Methods("POST").Path("/").HandlerFunc(b.handler)
	b.server = &http.Server{
		Addr:    BackendHost,
		Handler: b.httpHandler,
	}

	return &b
}

func (b *TestWebhook) handler(w http.ResponseWriter, r *http.Request) {
	p, _ := io.ReadAll(r.Body)
	msg := backend.BackendMessage{
		SessionId:    r.Header.Get(backend.SessionIdHeader),
		ReplyChannel: r.Header.Get(backend.ReplyChannelHeader),
		Event:        backend.ParseWsEvent(r.Header.Get(backend.EventHeader)),
		QueryString:  r.Header.Get(backend.QueryStringHeader),
		Payload:      p,
	}

	b.messages <- msg
	if len(b.responses) == 0 {
		w.WriteHeader(http.StatusNoContent)
	} else {
		resp := b.responses[0]
		b.responses = b.responses[1:]
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		w.Write(resp)
	}
}

func (b *TestWebhook) Start() {
	go func() {
		b.server.ListenAndServe()
	}()
}

func (b *TestWebhook) Stop() {
	b.server.Shutdown(context.Background())
}

func (b *TestWebhook) WaitForMessage(t *testing.T, timeout time.Duration) backend.BackendMessage {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case m := <-b.messages:
		return m
	case <-ctx.Done():
		t.Errorf("test backend should receive websocket message on time")
		panic("unreachable")
	}
}
