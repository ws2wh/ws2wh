package session

import (
	"testing"

	"github.com/pmartynski/ws2wh/backend"
	"github.com/stretchr/testify/assert"
)

// MockWebsocketConn implements WebsocketConn for testing
type MockWebsocketConn struct {
	sendCalled   bool
	closeCalled  bool
	receiverChan chan []byte
	doneChan     chan interface{}
	sendError    error
	closeError   error
}

func NewMockWebsocketConn() *MockWebsocketConn {
	return &MockWebsocketConn{
		receiverChan: make(chan []byte),
		doneChan:     make(chan interface{}),
	}
}

func (m *MockWebsocketConn) Send(payload []byte) error {
	m.sendCalled = true
	return m.sendError
}

func (m *MockWebsocketConn) Receiver() <-chan []byte {
	return m.receiverChan
}

func (m *MockWebsocketConn) Done() chan interface{} {
	return m.doneChan
}

func (m *MockWebsocketConn) Close() error {
	m.closeCalled = true
	return m.closeError
}

// MockBackend implements backend.Backend for testing
type MockBackend struct {
	messages []backend.BackendMessage
}

func (m *MockBackend) Send(msg backend.BackendMessage, s backend.SessionHandle) error {
	m.messages = append(m.messages, msg)
	return nil
}

func TestNewSession(t *testing.T) {
	conn := NewMockWebsocketConn()
	backend := &MockBackend{}

	params := SessionParams{
		Id:           "test-session",
		ReplyChannel: "http://test.com/reply",
		Backend:      backend,
		Connection:   conn,
	}

	session := NewSession(params)

	assert.Equal(t, params.Id, session.Id, "Session ID should match")
	assert.Equal(t, params.ReplyChannel, session.ReplyChannel, "Reply channel should match")
}

func TestSession_Send(t *testing.T) {
	conn := NewMockWebsocketConn()
	session := &Session{Connection: conn}

	message := []byte("test message")
	err := session.Send(message)

	assert.NoError(t, err, "Send should not return error")
	assert.True(t, conn.sendCalled, "Send should be called on WebsocketConn")
}

func TestSession_Close(t *testing.T) {
	conn := NewMockWebsocketConn()
	session := &Session{Connection: conn}

	err := session.Close()

	assert.NoError(t, err, "Close should not return error")
	assert.True(t, conn.closeCalled, "Close should be called on WebsocketConn")
}

func TestSession_Receive(t *testing.T) {
	conn := NewMockWebsocketConn()
	mockBackend := &MockBackend{}
	session := &Session{
		Id:           "test-session",
		ReplyChannel: "http://test.com/reply",
		Backend:      mockBackend,
		Connection:   conn,
	}

	// Test connection message
	go func() {
		// Simulate message received
		conn.receiverChan <- []byte("test message")
		// Then simulate connection close
		close(conn.doneChan)
	}()

	session.Receive()

	// Verify messages sent to backend
	assert.Len(t, mockBackend.messages, 3, "Should have 3 backend messages")

	// Verify connect message
	assert.Equal(t, backend.ClientConnected, mockBackend.messages[0].Event,
		"First message should be ClientConnected")

	// Verify received message
	assert.Equal(t, backend.MessageReceived, mockBackend.messages[1].Event,
		"Second message should be MessageReceived")
	assert.Equal(t, "test message", string(mockBackend.messages[1].Payload),
		"Message payload should match")

	// Verify disconnect message
	assert.Equal(t, backend.ClientDisconnected, mockBackend.messages[2].Event,
		"Last message should be ClientDisconnected")
}
