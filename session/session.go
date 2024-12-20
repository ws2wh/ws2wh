package session

import (
	"github.com/pmartynski/ws2wh/backend"
)

// Session represents a WebSocket session that bridges communication between a client and backend
type Session struct {
	// Id uniquely identifies this WebSocket session
	Id string
	// ReplyChannel is the URL where backend responses should be sent back to
	ReplyChannel string
	// Backend handles delivering messages to the configured backend service
	Backend backend.Backend
	// Connection manages the WebSocket connection with the client
	Connection WebsocketConn
}

// NewSession creates a new WebSocket session with the provided parameters
// params contains the session configuration including:
// - Id: Unique identifier for this session
// - ReplyChannel: URL where backend responses should be sent
// - Backend: Service for delivering messages to the backend
// - Connection: WebSocket connection manager for the client
// Returns a pointer to the newly created Session
func NewSession(params SessionParams) *Session {
	s := Session(params)

	return &s
}

// Send transmits a message through the WebSocket connection to the client
// message contains the raw bytes to send to the client
// Returns an error if sending the message fails
func (s *Session) Send(message []byte) error {
	return s.Connection.Send(message)
}

// Close terminates the WebSocket connection for this session
// Returns an error if closing the connection fails
func (s *Session) Close() error {
	return s.Connection.Close()
}

// Receive handles the WebSocket session lifecycle and message flow
// It performs the following:
// - Notifies the backend when a client connects
// - Forwards received messages from the client to the backend
// - Notifies the backend when the client disconnects
// - Cleans up the session when done
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

// WebsocketConn defines the interface for interacting with a WebSocket connection
// It provides methods for sending messages, receiving messages, checking connection status,
// and closing the connection
type WebsocketConn interface {
	Send(payload []byte) error
	Receiver() <-chan []byte
	Done() chan interface{}
	Close() error
}

// SessionParams contains the configuration parameters for creating a new Session
type SessionParams struct {
	// Id uniquely identifies this WebSocket session
	Id string
	// ReplyChannel is the URL where backend responses should be sent
	ReplyChannel string
	// Backend handles sending messages to the configured backend service
	Backend backend.Backend
	// Connection provides the WebSocket connection interface
	Connection WebsocketConn
}
