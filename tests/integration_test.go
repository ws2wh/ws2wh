package tests

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pmartynski/ws2wh/backend"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/websocket"
)

func TestWebsocketToWebhook(t *testing.T) {
	wsSrv := CreateTestWs()
	wsSrv.Start()
	defer wsSrv.Stop()

	wh := CreateTestWebhook()
	wh.Start()
	defer wh.Stop()

	// make sure ws server is up
	time.Sleep(time.Millisecond * 10)

	conn, sessionId, replyUrl := clientConnected(wh, t)
	websocketClientMessageSent(conn, wh, sessionId, t)
	backendMessageSent(conn, replyUrl, t)
	websocketClientMessageWithImmediateBackendResponse(conn, wh, sessionId, t)
	websocketClientDisconnected(conn, wh, sessionId, t)
}

func clientConnected(wh *TestWebhook, t *testing.T) (conn *websocket.Conn, sessionId string, replyUrl string) {
	assert := assert.New(t)

	conn, err := websocket.Dial(WsUrl, "", OriginUrl)
	assert.Nil(err, "should accept websocket connection")
	onConnected := wh.WaitForMessage(t)
	assert.NotNil(onConnected, "received message should not be nil")

	assert.Equal(backend.ClientConnected, onConnected.Event, "should receive on connected event message")
	sessionId = onConnected.SessionId
	replyUrl = onConnected.ReplyChannel
	return
}

func websocketClientMessageSent(conn *websocket.Conn, wh *TestWebhook, sessionId string, t *testing.T) {
	assert := assert.New(t)

	clientMsg := []byte(uuid.NewString())
	_, err := conn.Write(clientMsg)
	assert.Nil(err, "should successfully send websocket message via ws client")

	onMessage := wh.WaitForMessage(t)
	assert.Equal(sessionId, onMessage.SessionId, "backend should receive message with expected session id")
	assert.Equal(backend.MessageReceived, onMessage.Event, "backend should receive messagereceived message")
	assert.Equal(clientMsg, onMessage.Payload, "backend should receive exact same payload as the ws client sent in request body")
}

func backendMessageSent(conn *websocket.Conn, replyUrl string, t *testing.T) {
	assert := assert.New(t)

	wsClientChan := make(chan []byte, 1)
	defer close(wsClientChan)
	go captureMessage(conn, wsClientChan)

	expectedBackendMsg := []byte(uuid.NewString())
	resp, err := http.Post(replyUrl, "text/plain", bytes.NewReader(expectedBackendMsg))

	assert.Nil(err, "reply url call should not respond with client error")
	assert.Less(resp.StatusCode, 300, "reply url call should result with successful response")
	assert.GreaterOrEqual(resp.StatusCode, 200, "reply url call should result with successful response")

	actualBackendMsg := waitForMessage(t, wsClientChan)

	assert.Equal(expectedBackendMsg, actualBackendMsg, "reply url call body should be received by websocket client connected to session")
}

func websocketClientMessageWithImmediateBackendResponse(conn *websocket.Conn, wh *TestWebhook, sessionId string, t *testing.T) {
	assert := assert.New(t)
	expectedResponse := []byte(uuid.NewString())
	wh.responses = append(wh.responses, expectedResponse)
	wsClientChan := make(chan []byte, 1)
	defer close(wsClientChan)
	go captureMessage(conn, wsClientChan)
	websocketClientMessageSent(conn, wh, sessionId, t)
	actualResponse := waitForMessage(t, wsClientChan)
	assert.Equal(expectedResponse, actualResponse, "immediate backend response body should be received by websocket client connected to session")
}

func websocketClientDisconnected(conn *websocket.Conn, wh *TestWebhook, sessionId string, t *testing.T) {
	assert := assert.New(t)

	conn.Close()
	onClosed := wh.WaitForMessage(t)

	assert.NotNil(onClosed, "backend should receive non-empty message")
	assert.Equal(backend.ClientDisconnected, onClosed.Event, "backend received message should have client disconnected event header")
	assert.Equal(make([]byte, 0), onClosed.Payload, "backend received message should have an empty body")
	assert.Equal(sessionId, onClosed.SessionId, "backend received message should have proper session id header")
}

func captureMessage(ws *websocket.Conn, out chan []byte) {
	var incomingMsg []byte
	websocket.Message.Receive(ws, &incomingMsg)
	out <- incomingMsg
}

func waitForMessage(t *testing.T, out chan []byte) []byte {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	select {
	case m := <-out:
		return m
	case <-ctx.Done():
		t.Errorf("websocket client should receive backend message on time")
		panic("unreachable")
	}
}
