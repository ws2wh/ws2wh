package backend

import (
	"bytes"
	"fmt"
	"net/http"
)

type Backend interface {
	Send(payload *[]byte) error
}

type webhook struct {
	url         string
	contentType string
}

// Send implements Backend.
func (w webhook) Send(payload *[]byte) error {
	r, err := http.Post(w.url, w.contentType, bytes.NewReader(*payload))
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		var buffer []byte
		r.Body.Read(buffer)
		return fmt.Errorf("unsuccessful delivery %s\n%s", w.url, string(buffer))
	}

	return nil
}

func CreateBackend(url string) Backend {
	return webhook{
		url:         url,
		contentType: "application/json",
	}
}
