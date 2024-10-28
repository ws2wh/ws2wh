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
	assert := assert.New(t)
	wsSrv := CreateTestWs()
	wsSrv.Start()
	defer wsSrv.Stop()

	wh := CreateTestWebhook()
	wh.Start()
	defer wh.Stop()

	// make sure ws server is up
	time.Sleep(time.Millisecond * 10)

	conn, err := websocket.Dial(WsUrl, "", OriginUrl)
	assert.Nil(err)
	onConnected := wh.WaitForMessage(t)
	assert.NotNil(onConnected)

	assert.Equal(backend.ClientConnected, onConnected.Event)
	sessionId := onConnected.SessionId
	replyUrl := onConnected.ReplyChannel

	clientMsg := []byte(uuid.NewString())
	_, err = conn.Write(clientMsg)
	assert.Nil(err)

	onMessage := wh.WaitForMessage(t)
	assert.Equal(sessionId, onMessage.SessionId)
	assert.Equal(backend.MessageReceived, onMessage.Event)
	assert.Equal(clientMsg, onMessage.Payload)

	wsClientChan := make(chan []byte, 100)
	go captureMessage(conn, wsClientChan)
	expectedBackendMsg := []byte(uuid.NewString())
	resp, err := http.Post(replyUrl, "text/plain", bytes.NewReader(expectedBackendMsg))

	assert.Nil(err)
	assert.Less(resp.StatusCode, 300)
	assert.GreaterOrEqual(resp.StatusCode, 200)

	actualBackendMsg := waitForMessage(t, wsClientChan)

	assert.Equal(expectedBackendMsg, actualBackendMsg)

	conn.Close()
	onClosed := wh.WaitForMessage(t)
	assert.NotNil(onClosed)
	assert.Equal(backend.ClientDisconnected, onClosed.Event)
	assert.Equal(make([]byte, 0), onClosed.Payload)
	assert.Equal(sessionId, onClosed.SessionId)
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
		t.Errorf("Receiving message via ws client timed out")
		panic("unreachable")
	}
}
