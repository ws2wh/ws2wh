// Package frontend provides WebSocket connection handling functionality for ws2wh.
// It implements the WebSocket server-side logic for upgrading HTTP connections,
// managing message channels, and handling the WebSocket connection lifecycle.
package frontend

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	m "github.com/ws2wh/ws2wh/metrics/directory"
	"github.com/ws2wh/ws2wh/session"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewWsHandler creates a new WebsocketHandler with initialized channels
// for receiving messages and handling connection termination
func NewWsHandler(logger slog.Logger, id string) *WebsocketHandler {
	h := WebsocketHandler{
		receiverChannel: make(chan []byte, 64),
		signalChannel:   make(chan session.ConnectionSignal, 2),
		logger:          logger,
		sessionId:       id,
	}

	return &h
}

// WebsocketHandler manages a WebSocket connection and provides an interface
// for sending/receiving messages and handling connection lifecycle
type WebsocketHandler struct {
	receiverChannel chan []byte
	signalChannel   chan session.ConnectionSignal
	conn            *websocket.Conn
	logger          slog.Logger
	sessionId       string
	closed          bool
}

// Send writes a message to the WebSocket connection
func (h *WebsocketHandler) Send(data []byte) error {
	err := h.conn.WriteMessage(websocket.TextMessage, data)

	if err != nil {
		h.logger.Error("Error while sending message to client", "error", err, "sessionId", h.sessionId)
		m.MessageFailureCounter.With(prometheus.Labels{
			m.OriginLabel: m.OriginValueBackend,
		}).Inc()
	} else {
		m.MessageSuccessCounter.With(prometheus.Labels{
			m.OriginLabel: m.OriginValueBackend,
		}).Inc()
	}

	return err
}

// Receiver returns a channel for receiving incoming WebSocket messages
func (h *WebsocketHandler) Receiver() <-chan []byte {
	return h.receiverChannel
}

// Signal returns a channel that signals when the connection is ready or closed
func (h *WebsocketHandler) Signal() <-chan session.ConnectionSignal {
	return h.signalChannel
}

// Close gracefully terminates the WebSocket connection
func (h *WebsocketHandler) Close() error {
	h.signalChannel <- session.ConnectionClosedSignal

	h.closed = true

	err := h.conn.WriteMessage(websocket.CloseMessage, make([]byte, 0))
	if err != nil {
		return err
	}

	return h.conn.Close()
}

// Handle upgrades an HTTP connection to WebSocket and manages the connection lifecycle.
// It reads messages from the connection and forwards them to the receiver channel.
// The connection is terminated when a close message is received or on error.
func (h *WebsocketHandler) Handle(w http.ResponseWriter, r *http.Request, responseHeader http.Header) error {
	defer close(h.signalChannel)
	defer func() { h.signalChannel <- session.ConnectionClosedSignal }()
	defer close(h.receiverChannel)

	h.logger.Info("Upgrading HTTP to WS", "sessionId", h.sessionId)

	conn, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		h.logger.Error("Error while upgrading connection", "error", err, "sessionId", h.sessionId)
		return err
	}

	m.ConnectCounter.Inc()
	h.conn = conn
	h.signalChannel <- session.ConnectionReadySignal

	for {
		_, msg, err := conn.ReadMessage()

		if err != nil {
			return h.handleReadMessageErr(err)
		}

		h.logger.Debug("Received message", "sessionId", h.sessionId, "data", string(msg))
		h.receiverChannel <- msg
	}
}

func (h *WebsocketHandler) handleReadMessageErr(err error) error {
	if err == nil {
		return nil
	}

	defer func() { h.signalChannel <- session.ConnectionClosedSignal }()

	if h.closed {
		m.DisconnectCounter.With(prometheus.Labels{
			m.OriginLabel: m.OriginValueBackend,
		}).Inc()
		h.logger.Info("Backend closed connection", "sessionId", h.sessionId)
		return nil
	}

	if websocket.IsCloseError(err, 1000, 1001, 1005) {
		m.DisconnectCounter.With(prometheus.Labels{
			m.OriginLabel: m.OriginValueClient,
		}).Inc()

		h.logger.Info("Client closed connection", "sessionId", h.sessionId)
		return nil
	}

	m.DisconnectCounter.With(prometheus.Labels{
		m.OriginLabel: m.OriginValueClient,
	}).Inc()

	h.logger.Error("Error while reading message", "error", err, "sessionId", h.sessionId)

	return err
}
