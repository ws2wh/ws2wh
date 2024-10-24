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
	// TODO: create ws publish method
	s := Session{
		Id:     params.Id,
		params: params,
	}

	s.runReceiver()

	return &s
}

func (s *Session) runReceiver() {
	websocket.Handler(func(ws *websocket.Conn) {
		s.conn = ws
		for {
			var msg []byte
			e := websocket.Message.Receive(ws, &msg)
			if e != nil && e.Error() != "EOF" {
				log.Error(e)
			}
			if e != nil {
				break
			}

			s.params.Backend.Send(&msg)
		}
	}).ServeHTTP(s.params.Response, s.params.Request)
}

type SessionParams struct {
	Id      string
	Backend backend.Backend

	Request  *http.Request
	Response http.ResponseWriter
}
