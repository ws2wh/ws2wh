package backend

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
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
	wh := WebhookBackend{
		url:    "http://backend/wh/" + uuid.NewString(),
		client: &fc,
	}
	msg := BackendMessage{
		SessionId:    uuid.NewString(),
		ReplyChannel: "http://ws2wh-address/" + uuid.NewString(),
		Event:        MessageReceived,
		Payload:      []byte(uuid.NewString()),
	}

	sessionHandle := testSessionHandle{
		sendCount: 0,
	}
	err := wh.Send(msg, &sessionHandle)

	assert.Nil(err)
	assert.Len(fc.Requests, 1, "should receive 1 request")
	assert.Zero(sessionHandle.sendCount, "should not call the callback (empty wh response body)")
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
	wh := WebhookBackend{
		url:    "http://backend/wh/" + uuid.NewString(),
		client: &fc,
	}
	msg := BackendMessage{
		SessionId:    uuid.NewString(),
		ReplyChannel: "http://ws2wh-address/" + uuid.NewString(),
		Event:        MessageReceived,
		Payload:      []byte(uuid.NewString()),
	}

	sessionHandle := testSessionHandle{
		lastPayload: nil,
		sendCount:   0,
	}
	err := wh.Send(msg, &sessionHandle)

	assert.Nil(err)
	assert.Equal(1, sessionHandle.sendCount, "should call the callback once")
	assert.Equal(expectedPayload, sessionHandle.lastPayload, "should call the callback with response body payload")
}

func TestWebhookClientError(t *testing.T) {
	assert := assert.New(t)
	fc := fakeHttpClient{
		// sends error if no responses in the queue
		Responses: make([]*http.Response, 0),
	}
	wh := WebhookBackend{
		url:    "http://backend/wh/" + uuid.NewString(),
		client: &fc,
	}
	cbCount := 0

	err := wh.Send(BackendMessage{
		SessionId:    uuid.NewString(),
		ReplyChannel: "http://ws2wh-address/" + uuid.NewString(),
		Event:        ClientConnected,
		Payload:      []byte(uuid.NewString()),
	}, &testSessionHandle{})

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
	wh := WebhookBackend{
		url:    "http://backend/wh/" + uuid.NewString(),
		client: &fc,
	}
	cbCount := 0

	err := wh.Send(BackendMessage{
		SessionId:    uuid.NewString(),
		ReplyChannel: "http://ws2wh-address/" + uuid.NewString(),
		Event:        ClientDisconnected,
		Payload:      []byte(uuid.NewString()),
	}, &testSessionHandle{})

	assert.NotNil(err)
	assert.Zero(cbCount, "should not call the callback on HTTP error response")
}

func TestWebhookTerminateSessionWithCloseHeaders(t *testing.T) {
	assert := assert.New(t)

	fc := fakeHttpClient{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Status:     http.StatusText(200),
				Header: http.Header{
					CommandHeader:     []string{TerminateSessionCommand},
					CloseCodeHeader:   []string{"4000"},
					CloseReasonHeader: []string{"Shutting down"},
				},
				Body: io.NopCloser(bytes.NewReader(nil)),
			},
		},
	}
	wh := WebhookBackend{
		url:    "http://backend/wh/" + uuid.NewString(),
		client: &fc,
	}
	msg := BackendMessage{
		SessionId:    uuid.NewString(),
		ReplyChannel: "http://ws2wh-address/" + uuid.NewString(),
		Event:        MessageReceived,
		Payload:      []byte("payload"),
	}
	sh := testSessionHandle{}
	err := wh.Send(msg, &sh)
	assert.NoError(err)
	assert.Equal(1, sh.closeCount)
	assert.Equal(4000, sh.lastCloseCode)
	if assert.NotNil(sh.lastCloseReason) {
		assert.Equal("Shutting down", *sh.lastCloseReason)
	}
}

func TestGetCloseCode(t *testing.T) {
	validHeaderVals := []string{
		"1001",
		"",
	}

	expectedCloseCodes := []int{
		1001,
		1000,
	}

	for i, headerVal := range validHeaderVals {
		assert := assert.New(t)
		closeCode, err := GetCloseCode(headerVal)
		assert.Nil(err)
		assert.Equal(expectedCloseCodes[i], closeCode)
	}
}

func TestGetCloseCodeInvalid(t *testing.T) {
	invalidHeaderVals := []string{
		"999",
		"5000",
		"A",
		"1004",
		"1005",
		"1006",
		"1015",
	}

	for _, headerVal := range invalidHeaderVals {
		assert := assert.New(t)
		closeCode, err := GetCloseCode(headerVal)
		assert.NotNil(err)
		assert.Equal(0, closeCode)
	}
}

func TestGetCloseReason(t *testing.T) {
	validHeaderVals := []string{
		"",
		"test",
	}

	for _, headerVal := range validHeaderVals {
		assert := assert.New(t)
		closeReason, err := GetCloseReason(headerVal)
		assert.Nil(err)
		assert.Equal(headerVal, *closeReason)
	}
}

func TestGetCloseReasonInvalid(t *testing.T) {
	invalidHeaderVal := strings.Repeat("a", 124)

	assert := assert.New(t)
	closeReason, err := GetCloseReason(invalidHeaderVal)
	assert.NotNil(err)
	assert.Nil(closeReason)
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

type testSessionHandle struct {
	lastPayload     []byte
	sendCount       int
	closeCount      int
	lastCloseCode   int
	lastCloseReason *string
}

func (s *testSessionHandle) Send(payload []byte) error {
	s.lastPayload = payload
	s.sendCount += 1
	return nil
}
func (s *testSessionHandle) Close(closeCode int, closeReason *string) error {
	s.closeCount += 1
	s.lastCloseCode = closeCode
	s.lastCloseReason = closeReason
	return nil
}
