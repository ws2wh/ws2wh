// Package session provides functionality for managing WebSocket sessions in ws2wh.
// It handles the lifecycle of WebSocket connections, message routing between clients
// and the backend service, and session state management.
package session

import (
	"log/slog"

	"github.com/ws2wh/ws2wh/backend"
)

// Session represents a WebSocket session that bridges communication between a client and backend
type Session struct {
	// Id uniquely identifies this WebSocket session
	Id string
	// ReplyChannel is the URL where backend responses should be sent back to
	ReplyChannel string
	// QueryString contains the query string from the client
	QueryString string
	// Backend handles delivering messages to the configured backend service
	Backend backend.Backend
	// Connection manages the WebSocket connection with the client
	Connection WebsocketConn
	// Session logger
	Logger slog.Logger
	// JwtClaims contains the JWT payload from the client
	JwtClaims *string
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
	s.Logger.Debug("Sending message to client", "payload", string(message), "queryString", s.QueryString)

	return s.Connection.Send(message)
}

// Close terminates the WebSocket connection for this session
// Returns an error if closing the connection fails
func (s *Session) Close(closeCode int, closeReason *string) error {
	s.Logger.Debug("Closing WebSocket connection",
		"sessionId", s.Id,
		"closeCode", closeCode,
		"closeReason", closeReason,
	)
	return s.Connection.Close(closeCode, closeReason)
}

// Receive handles the WebSocket session lifecycle and message flow
// It performs the following:
// - Notifies the backend when a client connects
// - Forwards received messages from the client to the backend
// - Notifies the backend when the client disconnects
// - Cleans up the session when done
func (s *Session) Receive() {
	s.Logger.Debug("Waiting for connection signal")
	connSignal := <-s.Connection.Signal()
	s.Logger.Debug("Received connection signal", "signal", connSignal)
	if connSignal == ConnectionClosedSignal {
		s.Logger.Info("Session closed due to connection failure")
		return
	}

	if connSignal != ConnectionReadySignal {
		s.Logger.Error("Session closed due to unexpected connection signal", "signal", connSignal)
		return
	}

	s.Logger.Info("Starting WebSocket session", "sessionId", s.Id)

	msg := backend.BackendMessage{
		SessionId:    s.Id,
		ReplyChannel: s.ReplyChannel,
		Event:        backend.ClientConnected,
		Payload:      make([]byte, 0),
		QueryString:  s.QueryString,
		JwtClaims:    s.JwtClaims,
	}

	err := s.Backend.Send(msg, s)
	if err != nil {
		s.Logger.Error("Error while sending client connected message", "error", err)
	}
	msg.Event = backend.ClientDisconnected
	defer func() {
		s.Logger.Debug("Sending client disconnected message", "queryString", s.QueryString)
		err := s.Backend.Send(msg, s)
		if err != nil {
			s.Logger.Error("Error while sending client disconnected message", "error", err)
		}
	}()

loop:
	for {
		select {
		case incomingMsg := <-s.Connection.Receiver():
			s.Logger.Debug("Received message from client, forwarding to backend", "payload", string(incomingMsg), "queryString", s.QueryString)
			err := s.Backend.Send(backend.BackendMessage{
				SessionId:    s.Id,
				ReplyChannel: s.ReplyChannel,
				Event:        backend.MessageReceived,
				Payload:      incomingMsg,
				QueryString:  s.QueryString,
			}, s)
			if err != nil {
				s.Logger.Error("Error while sending message received message", "error", err)
			}
		case <-s.Connection.Signal():
			s.Logger.Info("Session done", "sessionId", s.Id)
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
	Signal() <-chan ConnectionSignal
	Close(closeCode int, closeReason *string) error
}

// SessionParams contains the configuration parameters for creating a new Session
type SessionParams struct {
	// Id uniquely identifies this WebSocket session
	Id string
	// ReplyChannel is the URL where backend responses should be sent
	ReplyChannel string
	// QueryString contains the query string from the client
	QueryString string
	// Backend handles sending messages to the configured backend service
	Backend backend.Backend
	// Connection provides the WebSocket connection interface
	Connection WebsocketConn
	// Session logger
	Logger slog.Logger
	// JwtClaims contains the JWT payload from the client
	JwtClaims *string
}

type ConnectionSignal int

const (
	// ConnectionReadySignal is the signal that the connection is ready
	ConnectionReadySignal ConnectionSignal = 1

	// ConnectionClosedSignal is the signal that the connection is closed
	ConnectionClosedSignal ConnectionSignal = 2
)
