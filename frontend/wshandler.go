package frontend

import (
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewWsHandler() *WebsocketHandler {

	h := WebsocketHandler{
		receiverChannel: make(chan []byte, 64),
		doneChannel:     make(chan interface{}, 1),
	}

	return &h
}

type WebsocketHandler struct {
	receiverChannel chan []byte
	doneChannel     chan interface{}
	conn            *websocket.Conn
}

func (h *WebsocketHandler) Send(data []byte) error {
	h.conn.WriteMessage(websocket.TextMessage, data)
	return nil
}

func (h *WebsocketHandler) Receiver() <-chan []byte {
	return h.receiverChannel
}

func (h *WebsocketHandler) Done() chan interface{} {
	return h.doneChannel
}

func (h *WebsocketHandler) Close() error {
	h.doneChannel <- 1
	h.conn.WriteMessage(websocket.CloseMessage, make([]byte, 0))
	defer h.conn.Close()

	return nil
}

func (h *WebsocketHandler) Handle(w http.ResponseWriter, r *http.Request, responseHeader http.Header) error {
	defer close(h.doneChannel)
	defer close(h.receiverChannel)

	conn, err := upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return err
	}

	h.conn = conn
	for {
		_, msg, err := conn.ReadMessage()
		if websocket.IsCloseError(err) {
			h.doneChannel <- 1
			return nil
		}

		if err != nil {
			h.doneChannel <- 1
			return err
		}

		h.receiverChannel <- msg
	}
}
