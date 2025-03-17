// Package server provides the HTTP and WebSocket server implementation for ws2wh
package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/ws2wh/ws2wh/backend"
	"github.com/ws2wh/ws2wh/frontend"
	m "github.com/ws2wh/ws2wh/metrics/directory"
	"github.com/ws2wh/ws2wh/session"
)

// Server handles WebSocket connections and forwards messages to a configured backend
type Server struct {
	DefaultBackend backend.Backend
	frontendAddr   string
	backendUrl     string
	replyUrl       string
	sessions       map[string]*session.Session
	echoStack      *echo.Echo
}

// CreateServer initializes a new Server instance with the given configuration
//
// Parameters:
//   - frontendAddr: The address and port to listen on (e.g. ":3000")
//   - websocketPath: The path where WebSocket connections will be upgraded (e.g. "/ws")
//   - backendUrl: The URL where backend messages will be sent
//   - replyPathPrefix: The prefix for reply endpoints (e.g. "/reply")
//   - replyUrl: The URL where reply messages will be sent (e.g. "http://my-host:3000/reply")
//
// Returns a configured Server instance ready to be started
func CreateServer(
	frontendAddr string,
	websocketPath string,
	backendUrl string,
	replyPathPrefix string,
	logLevel string,
	replyUrl string) *Server {

	s := Server{
		frontendAddr: frontendAddr,
		backendUrl:   backendUrl,
		replyUrl:     replyUrl,
		sessions:     make(map[string]*session.Session, 100),
	}

	es := echo.New()
	es.HideBanner = true
	es.HidePort = true
	es.Logger.SetLevel(parse(logLevel))

	s.DefaultBackend = backend.CreateBackend(backendUrl, es.Logger)

	es.Use(middleware.Logger())
	es.Use(middleware.Recover())

	replyPath := fmt.Sprintf("%s/:id", strings.TrimRight(replyPathPrefix, "/"))
	es.GET(websocketPath, s.handle)
	es.POST(replyPath, s.send)

	s.echoStack = es
	es.Logger.Infoj(map[string]interface{}{
		"message":       "Starting server...",
		"backendUrl":    backendUrl,
		"websocketPath": websocketPath,
		"frontendAddr":  frontendAddr,
	})

	return &s
}

// Start begins listening for connections on the configured address
func (s *Server) Start() {
	e := s.echoStack
	e.Logger.Errorj(map[string]interface{}{
		"error": e.Start(s.frontendAddr),
	})
}

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	err := s.echoStack.Shutdown(context.Background())
	if err != nil {
		s.echoStack.Logger.Fatalj(map[string]interface{}{
			"message": "Error while gracefully shutting down server",
			"error":   err,
		})
	}
}

func (s *Server) handle(c echo.Context) error {
	id := uuid.NewString()
	logger := c.Logger()
	handler := frontend.NewWsHandler(logger, id)

	s.sessions[id] = session.NewSession(session.SessionParams{
		Id:           id,
		Backend:      s.DefaultBackend,
		ReplyChannel: fmt.Sprintf("%s/%s", s.replyUrl, id),
		QueryString:  c.QueryString(),
		Connection:   handler,
		Logger:       logger,
	})

	m.ActiveSessionsGauge.Inc()

	defer m.ActiveSessionsGauge.Dec()
	defer delete(s.sessions, id)

	go s.sessions[id].Receive()
	err := handler.Handle(c.Response().Writer, c.Request(), c.Response().Header())
	if err != nil {
		c.Logger().Errorj(map[string]interface{}{
			"message": "Error while handling WebSocket connection",
			"error":   err,
		})
	}
	return nil
}

func (s *Server) send(c echo.Context) error {
	id := c.Param("id")
	var body []byte
	body, _ = io.ReadAll(c.Request().Body)
	session := s.sessions[id]

	if session == nil {
		err := c.JSON(http.StatusNotFound, SessionResponse{Success: false, Message: "NOT_FOUND"})
		if err != nil {
			c.Logger().Errorj(map[string]interface{}{
				"message": "Error while sending response",
				"error":   err,
			})
		}
	}

	if len(body) > 0 {
		err := session.Send(body)
		if err != nil {
			c.Logger().Errorj(map[string]interface{}{
				"message": "Error while sending message",
				"error":   err,
			})
		}
	}

	if c.Request().Header.Get(backend.CommandHeader) == backend.TerminateSessionCommand {
		err := session.Close()

		if err != nil {
			c.Logger().Errorj(map[string]interface{}{
				"message": "Error while closing session",
				"error":   err,
			})
		}
	}

	return c.JSON(http.StatusOK, SessionResponse{Success: true})
}

// SessionResponse represents the JSON response format for session-related operations
type SessionResponse struct {
	Success bool        `json:"success"`
	Message interface{} `json:"message,omitempty"`
}

func parse(logLevel string) log.Lvl {
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		return log.DEBUG
	case "INFO":
		return log.INFO
	case "WARN":
		return log.WARN
	case "ERROR":
		return log.ERROR
	case "OFF":
		return log.OFF
	}

	log.Warnf("Unknown log level: %s, using INFO instead", logLevel)
	return log.INFO
}
