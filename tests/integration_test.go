package tests

import (
	"bytes"
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
	wsServer := CreateWs()
	wsServer.Start()
	defer wsServer.Stop()

	testBackend := CreateBackend()
	testBackend.Start()
	defer testBackend.Stop()

	time.Sleep(time.Millisecond * 200)

	conn, err := websocket.Dial(WsUrl, "", OriginUrl)
	assert.Nil(err)
	onConnected := testBackend.WaitForMessage()
	assert.NotNil(onConnected)

	assert.Equal(backend.ClientConnected, onConnected.Event)
	sessionId := onConnected.SessionId
	replyUrl := onConnected.ReplyChannel

	clientMsg := []byte(uuid.NewString())
	_, err = conn.Write(clientMsg)
	assert.Nil(err)

	onMessage := testBackend.WaitForMessage()
	assert.Equal(sessionId, onMessage.SessionId)
	assert.Equal(backend.MessageReceived, onMessage.Event)
	assert.Equal(clientMsg, onMessage.Payload)

	wsChannel := make(chan []byte, 100)
	go WaitForMessage(conn, wsChannel)
	expectedBackendMsg := []byte(uuid.NewString())
	resp, err := http.Post(replyUrl, "text/plain", bytes.NewReader(expectedBackendMsg))

	assert.Nil(err)
	assert.Less(resp.StatusCode, 300)
	assert.GreaterOrEqual(resp.StatusCode, 200)

	actualBackendMsg := <-wsChannel
	assert.Equal(expectedBackendMsg, actualBackendMsg)
}
