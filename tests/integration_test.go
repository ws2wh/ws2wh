package tests

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pmartynski/ws2wh/backend"
	"github.com/stretchr/testify/assert"
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

	conn, _, replyUrl = clientConnected(wh, t)
	sessionTerminatedByBackend(conn, replyUrl, t)
}

func clientConnected(wh *TestWebhook, t *testing.T) (conn *websocket.Conn, sessionId string, replyUrl string) {
	assert := assert.New(t)

	conn, _, err := websocket.DefaultDialer.Dial(WsUrl, nil)
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
	err := conn.WriteMessage(websocket.TextMessage, clientMsg)
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

func sessionTerminatedByBackend(conn *websocket.Conn, replyUrl string, t *testing.T) {
	assert := assert.New(t)

	expectedGoodbyeMessage := []byte(uuid.NewString())

	req, _ := http.NewRequest(http.MethodPost, replyUrl, bytes.NewReader([]byte(expectedGoodbyeMessage)))

	req.Header = http.Header{
		backend.CommandHeader: {backend.TerminateSessionCommand},
	}

	messageType := make(chan int, 1)
	messageData := make(chan []byte, 1)
	var closed bool

	go func() {
		for {
			mt, d, err := conn.ReadMessage()
			if err != nil {
				break
			}
			messageType <- mt
			messageData <- d
		}

		closed = true
	}()

	r, e := http.DefaultClient.Do(req)
	assert.Nil(e, "backend should not get an http client error on calling reply url")
	assert.Equal(http.StatusOK, r.StatusCode, "backend should receive 200 on sending to reply url")

	mt := <-messageType
	actualGoodbyeMessage := <-messageData
	assert.Equal(websocket.TextMessage, mt)
	assert.Equal(expectedGoodbyeMessage, actualGoodbyeMessage)
	assert.True(closed)
}

func captureMessage(ws *websocket.Conn, out chan []byte) {
	_, incomingMsg, _ := ws.ReadMessage()
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
