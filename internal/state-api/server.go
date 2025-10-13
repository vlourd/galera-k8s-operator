package state_api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	agent "github.com/vlourd/galera-k8s-operator/internal/agent/entities"
	"github.com/vlourd/galera-k8s-operator/internal/entities"
	nodev1 "github.com/vlourd/galera-k8s-operator/internal/entities/node/betav1"
)

type Storage interface {
	GetGeneralResource(name, kind string) (entities.GeneralTargetState, error)
	UpdateGeneralResource(state entities.GeneralTargetState) error
	DeleteGeneralResource(name, kind string) error
	ListGeneralResourcesByKind(kind string) ([]entities.GeneralTargetState, error)
}

type Server struct {
	storage Storage
	l       *zap.Logger
	router  chi.Router
}

type TargetStateResponse struct {
	TargetState entities.GeneralTargetState `json:"target_state"`
	LastUpdated time.Time                   `json:"last_updated"`
}

func NewServer(l *zap.Logger, storage Storage) *Server {
	server := &Server{
		storage: storage,
		l:       l,
	}
	server.setupRoutes()

	return server
}

func (s *Server) setupRoutes() {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/heartbeat", s.handleHeartbeat)
		r.Get("/target-state", s.handleTargetState)
	})

	s.router = r
}

func (s *Server) Start(ctx context.Context, port string) error {
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      s.router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	// Handle shutdown in a goroutine
	go func() {
		<-ctx.Done() // Wait for context cancellation
		s.l.Info("Server shutting down...")

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			s.l.Error("Graceful shutdown failed", zap.Error(err))
		}
	}()

	s.l.Info("Server starting on port", zap.String("port", port))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server failed: %w", err)
	}

	s.l.Info("Server stopped")
	return nil
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req agent.State
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
		return
	}

	req.Timestamp = time.Now().UTC()
	s.l.Info("Heartbeat request", zap.Any("state", req))

	genNode, err := s.storage.GetGeneralResource(req.Name, nodev1.Kind)
	if err != nil {
		s.l.Error("Failed to get general resource", zap.Error(err))
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
	}

	if genNode.Name == "" {
		http.Error(w, "node not found", http.StatusNotFound)
	}

	node := nodev1.NodeTargetState{}
	err = node.FromGeneralTargetState(genNode)
	if err != nil {
		s.l.Error("Failed to get node from general resource", zap.Error(err))
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
	}

	node.Spec.Heartbeat = req
	genNode, err = node.ToGeneralTargetState()
	if err != nil {
		s.l.Error("Failed to get general resource", zap.Error(err))
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
	}

	err = s.storage.UpdateGeneralResource(genNode)
	if err != nil {
		s.l.Error("Failed to update general resource", zap.Error(err))
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
	}

	w.WriteHeader(http.StatusOK)
	s.l.Info("Heartbeat received", zap.Any("heartbeat", req), zap.String("node", req.Name))
}

func (s *Server) handleTargetState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Name parameter is required", http.StatusBadRequest)
		return
	}

	targetState, err := s.storage.GetGeneralResource(name, nodev1.Kind)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get target state: %v", err), http.StatusInternalServerError)
		return
	}

	if targetState.Name == "" {
		http.Error(w, "Target state not found", http.StatusNotFound)
		return
	}

	node := nodev1.NodeTargetState{}
	err = node.FromGeneralTargetState(targetState)
	if err != nil {
		s.l.Error("Failed to get node from general resource", zap.Error(err))
		http.Error(w, fmt.Sprintf("Bad request: %v", err), http.StatusBadRequest)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(node.Spec.AgentTargetState); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}
