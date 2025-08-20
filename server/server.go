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
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/ws2wh/ws2wh/backend"
	"github.com/ws2wh/ws2wh/frontend"
	"github.com/ws2wh/ws2wh/http-middleware/jwt"
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
	sessionsLock   sync.RWMutex
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
	s.DefaultBackend = backend.CreateBackend(config.BackendUrl)

	slog.Info("Starting server...",
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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Warn("Error during gracefully server shutdown", "err", err)
		}
	}()
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	id := uuid.NewString()
	handler := frontend.NewWsHandler(*slog.Default().With("sessionId", id), id)

	var jwtClaims *string
	if claims, ok := r.Context().Value(jwt.JwtClaimsKey{}).(map[string]interface{}); ok {
		if claimsJSON, err := json.Marshal(claims); err == nil {
			claimsStr := string(claimsJSON)
			jwtClaims = &claimsStr
		} else {
			slog.Error("Failed to marshal JWT claims", "error", err)
		}
	}

	s.addSession(id, session.NewSession(session.SessionParams{
		Id:           id,
		Backend:      s.DefaultBackend,
		ReplyChannel: fmt.Sprintf("%s/%s", s.replyUrl, id),
		QueryString:  r.URL.RawQuery,
		Connection:   handler,
		Logger:       *slog.Default().With("sessionId", id),
		JwtClaims:    jwtClaims,
	}))
	defer s.deleteSession(id)

	go func() {
		session := s.getSession(id)
		if session != nil {
			session.Receive()
		} else {
			slog.Warn("Session ended before starting to receive", "sessionId", id)
		}
	}()

	err := handler.Handle(w, r, w.Header())
	if err != nil {
		slog.Error("Error while handling WebSocket connection", "error", err)
	}
}

func (s *Server) getSession(id string) *session.Session {
	s.sessionsLock.RLock()
	defer s.sessionsLock.RUnlock()
	return s.sessions[id]
}

func (s *Server) deleteSession(id string) {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()
	delete(s.sessions, id)
	m.ActiveSessionsGauge.Dec()
}

func (s *Server) addSession(id string, session *session.Session) {
	s.sessionsLock.Lock()
	defer s.sessionsLock.Unlock()
	s.sessions[id] = session
	m.ActiveSessionsGauge.Inc()
}

func (s *Server) send(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	var body []byte
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Error reading request body", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(SessionResponse{Success: false, Message: "INVALID_REQUEST"})
		return
	}

	session := s.getSession(id)
	if session == nil {
		w.WriteHeader(http.StatusNotFound)
		err := json.NewEncoder(w).Encode(SessionResponse{Success: false, Message: "NOT_FOUND"})
		if err != nil {
			slog.Error("Error while sending response", "error", err)
		}
		return
	}

	if len(body) > 0 {
		err := session.Send(body)
		if err != nil {
			slog.Error("Error while sending message", "error", err)
		}
	}

	if r.Header.Get(backend.CommandHeader) == backend.TerminateSessionCommand {
		closeCode, err := backend.GetCloseCode(r.Header.Get(backend.CloseCodeHeader))
		if err != nil {
			slog.Error("Error while getting close code", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(SessionResponse{Success: false, Message: "INVALID_CLOSE_CODE"})
			return
		}

		closeReason, err := backend.GetCloseReason(r.Header.Get(backend.CloseReasonHeader))
		if err != nil {
			slog.Error("Error while getting close reason", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(SessionResponse{Success: false, Message: "INVALID_CLOSE_REASON"})
			return
		}

		err = session.Close(closeCode, closeReason)

		if err != nil {
			slog.Error("Error while closing session", "error", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(SessionResponse{Success: true})
	if err != nil {
		slog.Error("Error while sending response", "error", err)
	}
}

// SessionResponse represents the JSON response format for session-related operations
type SessionResponse struct {
	Success bool        `json:"success"`
	Message interface{} `json:"message,omitempty"`
}
