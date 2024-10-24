package server

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/pmartynski/ws2wh/backend"
	"github.com/pmartynski/ws2wh/session"
)

type Server struct {
	frontendAddr string
	backendUrl   string
}

func Run(frontendAddr string, backendUrl string) Server {
	s := Server{
		frontendAddr: frontendAddr,
		backendUrl:   backendUrl,
	}

	s.serve()
	return s
}

func (s *Server) serve() {
	log.SetLevel(log.DEBUG)
	e := echo.New()
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Recover())
	e.GET("/", s.handle)
	e.Logger.Fatal(e.Start(s.frontendAddr))
}

func (s *Server) handle(c echo.Context) error {
	session.NewSession(session.SessionParams{
		Id:      uuid.NewString(),
		Backend: backend.CreateBackend(s.backendUrl),

		Response: c.Response().Writer,
		Request:  c.Request(),
	})
	return nil
}

func (s *Server) Send(payload *[]byte) error {
	log.Debug("Sending: ", string((*payload)[:]))
	return nil
}
