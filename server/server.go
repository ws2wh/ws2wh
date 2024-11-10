package server

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/pmartynski/ws2wh/backend"
	"github.com/pmartynski/ws2wh/frontend"
	"github.com/pmartynski/ws2wh/session"
)

type Server struct {
	DefaultBackend backend.Backend
	frontendAddr   string
	backendUrl     string
	sessions       map[string]*session.Session
	echoStack      *echo.Echo
}

func CreateServer(frontendAddr string, backendUrl string) *Server {
	s := Server{
		frontendAddr: frontendAddr,
		backendUrl:   backendUrl,
		sessions:     make(map[string]*session.Session, 100),
	}
	s.DefaultBackend = backend.CreateBackend(backendUrl)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)

	// should we recover from panic?
	e.Use(middleware.Recover())
	e.GET("/", s.handle)
	e.POST("/reply/:id", s.send)

	s.echoStack = e

	return &s
}

func (s *Server) Start() {
	log.SetLevel(log.DEBUG)
	e := s.echoStack
	e.Logger.Info(e.Start(s.frontendAddr))
}

func (s *Server) Stop() {
	s.echoStack.Shutdown(context.Background())
}

func (s *Server) handle(c echo.Context) error {
	id := uuid.NewString()
	handler := frontend.NewWsHandler()

	s.sessions[id] = session.NewSession(session.SessionParams{
		Id:           id,
		Backend:      s.DefaultBackend,
		ReplyChannel: fmt.Sprintf("%s://%s/reply/%s", c.Scheme(), c.Request().Host, id),
		Connection:   handler,
	})

	defer delete(s.sessions, id)

	go s.sessions[id].Receive()
	handler.Handle(c.Response().Writer, c.Request(), c.Response().Header())
	return nil
}

func (s *Server) send(c echo.Context) error {
	id := c.Param("id")
	var body []byte
	body, _ = io.ReadAll(c.Request().Body)
	session := s.sessions[id]

	if session == nil {
		c.JSON(http.StatusNotFound, SessionResponse{Success: false, Message: "NOT_FOUND"})
	}

	if len(body) > 0 {
		session.Send(body)
	}

	if c.Request().Header.Get(backend.CommandHeader) == backend.TerminateSessionCommand {
		err := session.Close()

		if err != nil {
			c.Logger().Error(err)
		}
	}

	return c.JSON(http.StatusOK, SessionResponse{Success: true})
}

type SessionResponse struct {
	Success bool        `json:"success"`
	Message interface{} `json:"message,omitempty"`
}
