package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/pmartynski/ws2wh/backend"
	"github.com/pmartynski/ws2wh/session"
)

type Server struct {
	DefaultBackend backend.Backend
	frontendAddr   string
	backendUrl     string
	sessions       map[string]*session.Session
}

func CreateServer(frontendAddr string, backendUrl string) *Server {

	s := Server{
		frontendAddr: frontendAddr,
		backendUrl:   backendUrl,
		sessions:     make(map[string]*session.Session, 100),
	}
	s.DefaultBackend = backend.CreateBackend(backendUrl)

	return &s
}

func (s *Server) Serve() {
	log.SetLevel(log.DEBUG)
	e := echo.New()
	e.Use(middleware.Logger())
	e.Logger.SetLevel(log.DEBUG)

	// should we recover from panic?
	e.Use(middleware.Recover())
	e.GET("/", s.handle)
	e.POST("/:id", s.send)
	e.Logger.Fatal(e.Start(s.frontendAddr))
}

func (s *Server) handle(c echo.Context) error {
	id := uuid.NewString()

	s.sessions[id] = session.NewSession(session.SessionParams{
		Id:           id,
		Backend:      s.DefaultBackend,
		ReplyChannel: fmt.Sprintf("%s://%s/%s", c.Scheme(), c.Request().Host, id),
		Response:     c.Response().Writer,
		Request:      c.Request(),
	})
	defer delete(s.sessions, id)

	s.sessions[id].RunReceiver()
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

	session.Send(body)

	return c.JSON(http.StatusOK, SessionResponse{Success: true})
}

type SessionResponse struct {
	Success bool        `json:"success"`
	Message interface{} `json:"message,omitempty"`
}
