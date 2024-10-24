package session

import (
	"net/http"

	"github.com/labstack/gommon/log"
	"github.com/pmartynski/ws2wh/backend"
	"golang.org/x/net/websocket"
)

type Session struct {
	Id     string
	params SessionParams
	conn   *websocket.Conn
}

func NewSession(params SessionParams) *Session {
	s := Session{
		Id:     params.Id,
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
		s.conn = ws
		backend := *s.params.Backend
		backend.Send(s.Id, []byte("Session open"))
		defer backend.Send(s.Id, []byte("Session closed"))

		for {
			var msg []byte
			e := websocket.Message.Receive(ws, &msg)
			if e != nil && e.Error() != "EOF" {
				log.Error(e)
			}
			if e != nil {
				break
			}

			backend.Send(s.Id, msg)
		}
	}).ServeHTTP(s.params.Response, s.params.Request)
}

type SessionParams struct {
	Id      string
	Backend *backend.Backend

	Request  *http.Request
	Response http.ResponseWriter
}
