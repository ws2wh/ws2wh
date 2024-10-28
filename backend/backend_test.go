package backend

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWebhookSuccess(t *testing.T) {
	assert := assert.New(t)

	fc := fakeClient{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(200),
			},
		},
	}
	wh := webhook{
		url:    "http://backend/wh",
		client: &fc,
	}

	msg := BackendMessage{
		SessionId:    "ccj12cascdj10c910jc9",
		ReplyChannel: "http://ws2wh-address/who2033cas",
		Event:        MessageReceived,
		Payload:      []byte("HELLO"),
	}
	err := wh.Send(msg)

	assert.Nil(err)
	assert.Len(fc.Requests, 1)
	req := fc.Requests[0]

	assert.Equal(wh.url, req.URL.String())
	assert.Equal(msg.SessionId, req.Header.Get(SessionIdHeader))
	assert.Equal(msg.ReplyChannel, req.Header.Get(ReplyChannelHeader))
	assert.Equal(MessageReceived.String(), req.Header.Get(EventHeader))
	body, err := io.ReadAll(req.Body)
	assert.Nil(err)
	assert.Equal(string(msg.Payload), string(body))
}

func TestWebhookClientError(t *testing.T) {
	assert := assert.New(t)
	fc := fakeClient{
		Responses: make([]*http.Response, 0),
	}
	wh := webhook{
		url:    "http://backend/wh",
		client: &fc,
	}

	err := wh.Send(BackendMessage{
		SessionId:    "ccj12cascdj10c910jc9",
		ReplyChannel: "http://ws2wh-address/who2033cas",
		Event:        ClientConnected,
		Payload:      []byte("HELLO"),
	})

	assert.NotNil(err)
}

func TestWebhookServiceError(t *testing.T) {
	assert := assert.New(t)
	fc := fakeClient{
		Responses: []*http.Response{
			{
				Status:     http.StatusText(http.StatusTooManyRequests),
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(bytes.NewReader([]byte("TooManyRequests"))),
			},
		},
	}
	wh := webhook{
		url:    "http://backend/wh",
		client: &fc,
	}

	err := wh.Send(BackendMessage{
		SessionId:    "ccj12cascdj10c910jc9",
		ReplyChannel: "http://ws2wh-address/who2033cas",
		Event:        ClientDisconnected,
		Payload:      []byte("HELLO"),
	})

	assert.NotNil(err)
}

type fakeClient struct {
	Requests  []*http.Request
	Responses []*http.Response
}

func (c *fakeClient) Do(req *http.Request) (*http.Response, error) {
	c.Requests = append(c.Requests, req)
	if len(c.Responses) == 0 {
		return nil, &url.Error{
			URL: req.RequestURI,
		}
	}

	if len(c.Responses) == 1 {
		r := c.Responses[0]
		c.Responses = make([]*http.Response, 0)
		return r, nil
	}

	head, tail := c.Responses[0], c.Responses[1:]
	c.Responses = tail
	return head, nil
}
