package backend

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type Backend interface {
	Send(sessionId string, payload []byte) error
}

type webhook struct {
	url         string
	contentType string
	client      *http.Client
}

// Send implements Backend.
func (w *webhook) Send(sessionId string, payload []byte) error {
	req, err := http.NewRequest(http.MethodPost, w.url, bytes.NewReader(payload))
	h := http.Header{
		"WS-SessionId":    {sessionId},
		"WS-ClientChanel": {"http://localhost:3000/" + sessionId},
	}
	req.Header = h

	if err != nil {
		return err
	}

	res, err := w.client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("unsuccessful delivery %s\n%s", w.url, string(body))
	}

	return nil
}

func CreateBackend(url string) Backend {
	return &webhook{
		url:         url,
		contentType: "application/json",
		client:      http.DefaultClient,
	}
}
