package backend

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWebhookSuccess(t *testing.T) {
	assert := assert.New(t)
	fc := fakeHttpClient{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(200),
				Body:       io.NopCloser(bytes.NewReader(make([]byte, 0))),
			},
		},
	}
	wh := webhookBackend{
		url:    "http://backend/wh/" + uuid.NewString(),
		client: &fc,
	}
	msg := BackendMessage{
		SessionId:    uuid.NewString(),
		ReplyChannel: "http://ws2wh-address/" + uuid.NewString(),
		Event:        MessageReceived,
		Payload:      []byte(uuid.NewString()),
	}
	cbCount := 0

	err := wh.Send(msg, func(b []byte) { cbCount += 1 })

	assert.Nil(err)
	assert.Len(fc.Requests, 1, "should receive 1 request")
	assert.Zero(cbCount, "should not call the callback (empty wh response body)")
	req := fc.Requests[0]

	assert.Equal(wh.url, req.URL.String(), "should request configured webhook url")
	assert.Equal(msg.SessionId, req.Header.Get(SessionIdHeader), "request should conain session id header")
	assert.Equal(msg.ReplyChannel, req.Header.Get(ReplyChannelHeader), "request should contain reply channel header")
	assert.Equal(MessageReceived.String(), req.Header.Get(EventHeader), "request should contain event name header")

	body, err := io.ReadAll(req.Body)
	assert.Nil(err)
	assert.Equal(msg.Payload, body, "request body should be same WH message payload")
}

func TestWebhookSuccessWithPayload(t *testing.T) {
	assert := assert.New(t)
	expectedPayload := []byte(uuid.NewString())
	fc := fakeHttpClient{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(200),
				Body:       io.NopCloser(bytes.NewReader(expectedPayload)),
			},
		},
	}
	wh := webhookBackend{
		url:    "http://backend/wh/" + uuid.NewString(),
		client: &fc,
	}
	msg := BackendMessage{
		SessionId:    uuid.NewString(),
		ReplyChannel: "http://ws2wh-address/" + uuid.NewString(),
		Event:        MessageReceived,
		Payload:      []byte(uuid.NewString()),
	}
	cbCount := 0

	var actualPayload []byte
	err := wh.Send(msg, func(b []byte) { cbCount += 1; actualPayload = b })

	assert.Nil(err)
	assert.Equal(1, cbCount, "should call the callback once")
	assert.Equal(expectedPayload, actualPayload, "should call the callback with response body payload")
}

func TestWebhookClientError(t *testing.T) {
	assert := assert.New(t)
	fc := fakeHttpClient{
		// sends error if no responses in the queue
		Responses: make([]*http.Response, 0),
	}
	wh := webhookBackend{
		url:    "http://backend/wh/" + uuid.NewString(),
		client: &fc,
	}
	cbCount := 0

	err := wh.Send(BackendMessage{
		SessionId:    uuid.NewString(),
		ReplyChannel: "http://ws2wh-address/" + uuid.NewString(),
		Event:        ClientConnected,
		Payload:      []byte(uuid.NewString()),
	}, func(b []byte) { cbCount += 1 })

	assert.NotNil(err)
	assert.Zero(cbCount, "should not call the callback on client error")
}

func TestWebhookServiceError(t *testing.T) {
	assert := assert.New(t)
	fc := fakeHttpClient{
		Responses: []*http.Response{
			{
				Status:     http.StatusText(http.StatusTooManyRequests),
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(bytes.NewReader([]byte("TooManyRequests"))),
			},
		},
	}
	wh := webhookBackend{
		url:    "http://backend/wh/" + uuid.NewString(),
		client: &fc,
	}
	cbCount := 0

	err := wh.Send(BackendMessage{
		SessionId:    uuid.NewString(),
		ReplyChannel: "http://ws2wh-address/" + uuid.NewString(),
		Event:        ClientDisconnected,
		Payload:      []byte(uuid.NewString()),
	}, func(b []byte) { cbCount += 1 })

	assert.NotNil(err)
	assert.Zero(cbCount, "should not call the callback on HTTP error response")
}

type fakeHttpClient struct {
	Requests  []*http.Request
	Responses []*http.Response
}

func (c *fakeHttpClient) Do(req *http.Request) (*http.Response, error) {
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
