package backend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

const SessionIdHeader = "Ws-Session-Id"
const ReplyChannelHeader = "Ws-Reply-Channel"
const EventHeader = "Ws-Event"
const CommandHeader = "Ws-Command"

const SendMessageCommand = "send-message"
const TerminateSessionCommand = "terminate-session"

type WsEvent int

const (
	Unknown WsEvent = iota
	ClientConnected
	MessageReceived
	ClientDisconnected
)

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

type Backend interface {
	Send(msg BackendMessage, session SessionHandle) error
}

func CreateBackend(url string) Backend {
	return &webhookBackend{
		url:    url,
		client: http.DefaultClient,
	}
}

type BackendMessage struct {
	SessionId    string
	ReplyChannel string
	Event        WsEvent
	Payload      []byte
}

type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type webhookBackend struct {
	url    string
	client HttpClient
}

func (w *webhookBackend) Send(msg BackendMessage, session SessionHandle) error {
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

type SessionHandle interface {
	Send(message []byte) error
	Close() error
}
