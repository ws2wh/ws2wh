package backend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// SessionIdHeader is used to identify the WebSocket session in HTTP headers
const SessionIdHeader = "Ws-Session-Id"

// ReplyChannelHeader contains the URL where webhook responses should be sent
const ReplyChannelHeader = "Ws-Reply-Channel"

// EventHeader indicates the type of WebSocket event that occurred
const EventHeader = "Ws-Event"

// CommandHeader specifies the command to execute on the WebSocket connection
const CommandHeader = "Ws-Command"

// SendMessageCommand instructs the server to send a message to the WebSocket client
const SendMessageCommand = "send-message"

// TerminateSessionCommand instructs the server to close the WebSocket connection
const TerminateSessionCommand = "terminate-session"

// WsEvent represents different types of WebSocket events that can occur
type WsEvent int

const (
	// Unknown represents an unrecognized WebSocket event
	Unknown WsEvent = iota
	// ClientConnected indicates a new WebSocket client has connected
	ClientConnected
	// MessageReceived indicates a message was received from a WebSocket client
	MessageReceived
	// ClientDisconnected indicates a WebSocket client has disconnected
	ClientDisconnected
)

// String returns the string representation of a WsEvent
// It converts the WsEvent enum value to its corresponding string name:
// - ClientConnected -> "client-connected"
// - MessageReceived -> "message-received"
// - ClientDisconnected -> "client-disconnected"
// - Unknown/default -> "unknown"
func (e WsEvent) String() string {
	switch e {
	case ClientConnected:
		return "client-connected"
	case MessageReceived:
		return "message-received"
	case ClientDisconnected:
		return "client-disconnected"
	default:
		return "unknown"
	}
}

// ParseWsEvent converts a string event name to its corresponding WsEvent enum value
// It maps the following strings to WsEvent values:
// - "client-connected" -> ClientConnected
// - "message-received" -> MessageReceived
// - "client-disconnected" -> ClientDisconnected
// - Any other string -> Unknown
func ParseWsEvent(e string) WsEvent {
	switch e {
	case "client-connected":
		return ClientConnected
	case "message-received":
		return MessageReceived
	case "client-disconnected":
		return ClientDisconnected
	default:
		return Unknown
	}
}

// Backend defines the interface for sending messages to a backend service
// It provides a single method Send() for delivering messages to the configured backend
type Backend interface {
	// Send delivers a message to the backend service
	// msg contains the message details including session ID, reply channel, event type and payload
	// session provides a handle to the WebSocket session for sending responses
	// Returns an error if the message delivery fails
	Send(msg BackendMessage, session SessionHandle) error
}

// CreateBackend creates a new Backend instance that sends messages via HTTP webhooks
// url specifies the webhook endpoint URL that will receive the messages
// Returns a Backend interface using the default HTTP client for making webhook requests
func CreateBackend(url string) *WebhookBackend {
	return &WebhookBackend{
		url:    url,
		client: http.DefaultClient,
	}
}

// BackendMessage represents a message to be sent to the backend service
type BackendMessage struct {
	// SessionId uniquely identifies the WebSocket session this message belongs to
	SessionId string
	// ReplyChannel is the URL where responses should be sent back to
	ReplyChannel string
	// Event indicates what type of WebSocket event triggered this message
	Event WsEvent
	// Payload contains the raw message data bytes
	Payload []byte
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type WebhookBackend struct {
	url    string
	client httpClient
}

// Send delivers a message to the configured webhook endpoint
// It sends the message payload and headers via HTTP POST and handles the response
// msg contains the message details including session ID, reply channel, event type and payload
// session provides a handle to send responses back through the WebSocket connection
// Returns an error if the request fails or receives a non-2xx response
func (w *WebhookBackend) Send(msg BackendMessage, session SessionHandle) error {
	req, err := http.NewRequest(http.MethodPost, w.url, bytes.NewReader(msg.Payload))
	h := http.Header{
		SessionIdHeader:    {msg.SessionId},
		ReplyChannelHeader: {msg.ReplyChannel},
		EventHeader:        {msg.Event.String()},
	}
	req.Header = h

	if err != nil {
		return err
	}

	res, err := w.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("unsuccessful delivery to %s", w.url)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if len(body) > 0 && msg.Event != ClientDisconnected {
		session.Send(body)
	}

	if res.Header.Get(CommandHeader) == TerminateSessionCommand {
		session.Close()
	}
	return nil
}

// SessionHandle provides an interface for interacting with a WebSocket session
type SessionHandle interface {
	// Send transmits a message through the WebSocket connection
	// message is the payload to send to the client
	// Returns an error if the send fails
	Send(message []byte) error

	// Close terminates the WebSocket session
	// Returns an error if the close fails
	Close() error
}
