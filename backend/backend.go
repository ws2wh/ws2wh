package backend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

const SessionIdHeader = "WS-Session-Id"
const ReplyChannelHeader = "WS-Reply-Channel"
const EventHeader = "Ws-Event"

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
		return "ClientConnected"
	case MessageReceived:
		return "MessageReceived"
	case ClientDisconnected:
		return "ClientDisconnected"
	default:
		return "Uknown"
	}
}

type Backend interface {
	Send(msg WsMessage) error
}

func CreateBackend(url string) Backend {
	return &webhook{
		url:    url,
		client: http.DefaultClient,
	}
}

type WsMessage struct {
	SessionId    string
	ReplyChannel string
	Event        WsEvent
	Payload      []byte
}

type webhook struct {
	url    string
	client *http.Client
}

func (w *webhook) Send(msg WsMessage) error {
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

	if res.StatusCode >= 200 && res.StatusCode < 300 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("unsuccessful delivery %s\n%s", w.url, string(body))
	}

	return nil
}
