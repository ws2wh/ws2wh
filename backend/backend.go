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
	Send(msg BackendMessage) error
}

func CreateBackend(url string) Backend {
	return &webhook{
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

type webhook struct {
	url    string
	client HttpClient
}

func (w *webhook) Send(msg BackendMessage) error {
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

	return nil
}
