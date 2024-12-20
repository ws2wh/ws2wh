// Package frontend provides WebSocket connection handling functionality for ws2wh.
// It implements the WebSocket server-side logic for upgrading HTTP connections,
// managing message channels, and handling the WebSocket connection lifecycle.
package frontend

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewWsHandler creates a new WebsocketHandler with initialized channels
// for receiving messages and handling connection termination
func NewWsHandler() *WebsocketHandler {
	h := WebsocketHandler{
		receiverChannel: make(chan []byte, 64),
		doneChannel:     make(chan interface{}, 1),
	}

	return &h
}

// WebsocketHandler manages a WebSocket connection and provides an interface
// for sending/receiving messages and handling connection lifecycle
type WebsocketHandler struct {
	receiverChannel chan []byte
	doneChannel     chan interface{}
	conn            *websocket.Conn
}

// Send writes a message to the WebSocket connection
func (h *WebsocketHandler) Send(data []byte) error {
	h.conn.WriteMessage(websocket.TextMessage, data)
	return nil
}

// Receiver returns a channel for receiving incoming WebSocket messages
func (h *WebsocketHandler) Receiver() <-chan []byte {
	return h.receiverChannel
}

// Done returns a channel that signals when the connection is terminated
func (h *WebsocketHandler) Done() chan interface{} {
	return h.doneChannel
}

// Close gracefully terminates the WebSocket connection
func (h *WebsocketHandler) Close() error {
	h.doneChannel <- 1
	h.conn.WriteMessage(websocket.CloseMessage, make([]byte, 0))
	defer h.conn.Close()

	return nil
}

// Handle upgrades an HTTP connection to WebSocket and manages the connection lifecycle.
// It reads messages from the connection and forwards them to the receiver channel.
// The connection is terminated when a close message is received or on error.
func (h *WebsocketHandler) Handle(w http.ResponseWriter, r *http.Request, responseHeader http.Header) error {
	defer close(h.doneChannel)
	defer close(h.receiverChannel)

	conn, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return err
	}

	h.conn = conn
	for {
		_, msg, err := conn.ReadMessage()
		if websocket.IsCloseError(err) {
			h.doneChannel <- 1
			return nil
		}

		if err != nil {
			h.doneChannel <- 1
			return err
		}

		h.receiverChannel <- msg
	}
}
