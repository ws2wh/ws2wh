// Package server provides the HTTP and WebSocket server implementation for ws2wh
package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/ws2wh/ws2wh/backend"
	"github.com/ws2wh/ws2wh/frontend"
	m "github.com/ws2wh/ws2wh/metrics/directory"
	"github.com/ws2wh/ws2wh/session"
)

// TODO: move to new package, set default logger
var logHandler = slog.NewJSONHandler(os.Stdout, nil)
var logger = slog.New(logHandler)

// Server handles WebSocket connections and forwards messages to a configured backend
type Server struct {
	DefaultBackend backend.Backend
	frontendAddr   string
	backendUrl     string
	replyUrl       string
	sessions       map[string]*session.Session
	httpHandler    http.Handler
	tlsCertPath    string
	tlsKeyPath     string
}

// CreateServerWithConfig initializes a new Server instance with the given configuration
//
// Parameters:
//   - config: A pointer to a Config struct containing the server configuration
//
// # Returns a configured Server instance ready to be started
func CreateServerWithConfig(config *Config) *Server {
	s := Server{
		frontendAddr: config.WebSocketListener,
		backendUrl:   config.BackendUrl,
		replyUrl:     config.ReplyChannelConfig.GetReplyUrl(),
		sessions:     make(map[string]*session.Session, 100),
		tlsCertPath:  config.TlsConfig.TlsCertPath,
		tlsKeyPath:   config.TlsConfig.TlsKeyPath,
	}

	s.initMux(config)
	s.DefaultBackend = backend.CreateBackend(config.BackendUrl, *slog.New(logHandler))

	logger.Info("Starting server...",
		"backendUrl", config.BackendUrl,
		"websocketPath", config.WebSocketPath,
		"frontendAddr", config.WebSocketListener,
	)

	return &s
}

func (s *Server) initMux(config *Config) {
	router := mux.NewRouter()
	router.Path(config.WebSocketPath).Methods("GET").HandlerFunc(s.handle)
	replyPath := fmt.Sprintf("%s/{id}", strings.TrimRight(config.ReplyChannelConfig.PathPrefix, "/"))
	router.Path(replyPath).Methods("POST").HandlerFunc(s.send)

	s.httpHandler = router
}

// func (s *Server) buildEchoStack(config *Config) http.Handler {
// 	// TODO: replace echo stack with gorilla mux
// 	es := echo.New()
// 	es.HideBanner = true
// 	es.HidePort = true
// 	es.Logger.SetLevel(config.LogLevel)

// 	es.Use(middleware.Logger())
// 	es.Use(middleware.Recover())

// 	replyPath := fmt.Sprintf("%s/:id", strings.TrimRight(config.ReplyChannelConfig.PathPrefix, "/"))
// 	es.GET(config.WebSocketPath, s.handle)
// 	es.POST(replyPath, s.send)

// 	return es
// }

// Start begins listening for connections on the configured address
func (s *Server) Start(ctx context.Context) {
	server := &http.Server{
		Addr: s.frontendAddr,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		Handler: s.httpHandler,
	}

	go func() {
		var err error
		if s.tlsCertPath != "" && s.tlsKeyPath != "" {
			err = server.ListenAndServeTLS(s.tlsCertPath, s.tlsKeyPath)
		} else {
			err = server.ListenAndServe()
		}

		if err != nil {
			slog.Error("Http server stopped", "err", err)
		}
	}()

	go func() {
		<-ctx.Done()
		if err := server.Shutdown(context.Background()); err != nil {
			slog.Error("Error during gracefully server shutdown", "err", err)
		}
	}()
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	id := uuid.NewString()
	handler := frontend.NewWsHandler(*logger, id)

	s.sessions[id] = session.NewSession(session.SessionParams{
		Id:           id,
		Backend:      s.DefaultBackend,
		ReplyChannel: fmt.Sprintf("%s/%s", s.replyUrl, id),
		QueryString:  r.URL.RawQuery,
		Connection:   handler,
		Logger:       *logger,
	})

	m.ActiveSessionsGauge.Inc()

	defer m.ActiveSessionsGauge.Dec()
	defer delete(s.sessions, id)

	go s.sessions[id].Receive()
	err := handler.Handle(w, r, w.Header())
	if err != nil {
		logger.Error("Error while handling WebSocket connection", "error", err)
	}
}

func (s *Server) send(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var body []byte
	body, _ = io.ReadAll(r.Body)
	session := s.sessions[id]

	if session == nil {
		w.WriteHeader(http.StatusNotFound)
		err := json.NewEncoder(w).Encode(SessionResponse{Success: false, Message: "NOT_FOUND"})
		if err != nil {
			logger.Error("Error while sending response", "error", err)
		}
		return
	}

	if len(body) > 0 {
		err := session.Send(body)
		if err != nil {
			logger.Error("Error while sending message", "error", err)
		}
	}

	if r.Header.Get(backend.CommandHeader) == backend.TerminateSessionCommand {
		err := session.Close()

		if err != nil {
			logger.Error("Error while closing session", "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(SessionResponse{Success: true})
	if err != nil {
		logger.Error("Error while sending response", "error", err)
	}
}

// SessionResponse represents the JSON response format for session-related operations
type SessionResponse struct {
	Success bool        `json:"success"`
	Message interface{} `json:"message,omitempty"`
}
