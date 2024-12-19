package session

import (
	"github.com/pmartynski/ws2wh/backend"
)

type Session struct {
	Id           string
	ReplyChannel string
	Backend      backend.Backend
	Connection   WebsocketConn
}

func NewSession(params SessionParams) *Session {
	s := Session(params)

	return &s
}

func (s *Session) Send(message []byte) error {
	return s.Connection.Send(message)
}

func (s *Session) Close() error {
	return s.Connection.Close()
}

func (s *Session) Receive() {

	msg := backend.BackendMessage{
		SessionId:    s.Id,
		ReplyChannel: s.ReplyChannel,
		Event:        backend.ClientConnected,
		Payload:      make([]byte, 0),
	}

	s.Backend.Send(msg, s)
	msg.Event = backend.ClientDisconnected
	defer s.Backend.Send(msg, s)

loop:
	for {
		select {
		case incomingMsg := <-s.Connection.Receiver():
			s.Backend.Send(backend.BackendMessage{
				SessionId:    s.Id,
				ReplyChannel: s.ReplyChannel,
				Event:        backend.MessageReceived,
				Payload:      incomingMsg,
			}, s)
		case <-s.Connection.Done():
			break loop
		}
	}
}

type WebsocketConn interface {
	Send(payload []byte) error
	Receiver() <-chan []byte
	Done() chan interface{}
	Close() error
}

type SessionParams struct {
	Id           string
	ReplyChannel string
	Backend      backend.Backend
	Connection   WebsocketConn
}
