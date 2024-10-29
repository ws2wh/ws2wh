package session

import (
	"net/http"

	"github.com/labstack/gommon/log"
	"github.com/pmartynski/ws2wh/backend"
	"golang.org/x/net/websocket"
)

type Session struct {
	params SessionParams
	conn   *websocket.Conn
}

func NewSession(params SessionParams) *Session {
	s := Session{
		params: params,
	}

	return &s
}

func (s *Session) Send(message []byte) error {
	_, e := s.conn.Write(message)
	return e
}

func (s *Session) RunReceiver() {
	websocket.Handler(func(ws *websocket.Conn) {
		callback := func(b []byte) {
			err := s.Send(b)
			if err != nil {
				log.Error(err)
			}
		}
		s.conn = ws
		b := s.params.Backend
		msg := backend.BackendMessage{
			SessionId:    s.params.Id,
			ReplyChannel: s.params.ReplyChannel,
			Event:        backend.ClientConnected,
			Payload:      make([]byte, 0),
		}
		b.Send(msg, callback)

		msg.Event = backend.ClientDisconnected
		defer b.Send(msg, func(b []byte) {})

		for {
			var incomingMsg []byte
			e := websocket.Message.Receive(ws, &incomingMsg)
			if e != nil && e.Error() != "EOF" {
				log.Error(e)
			}
			if e != nil {
				break
			}

			b.Send(backend.BackendMessage{
				SessionId:    s.params.Id,
				ReplyChannel: s.params.ReplyChannel,
				Event:        backend.MessageReceived,
				Payload:      incomingMsg,
			}, callback)
		}
	}).ServeHTTP(s.params.Response, s.params.Request)
}

type SessionParams struct {
	Id           string
	ReplyChannel string
	Backend      backend.Backend

	Request  *http.Request
	Response http.ResponseWriter
}
