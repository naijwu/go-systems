package ui

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

type StatusServer struct {
	server  *http.Server
	service *StatusService
	logger  *log.Logger
}

func NewStatusServer(addr string, service *StatusService, logger *log.Logger) *StatusServer {
	mux := http.NewServeMux()

	s := &StatusServer{
		server: &http.Server{
			Addr:              addr,
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		},
		service: service,
		logger:  logger,
	}

	mux.HandleFunc("/status", s.handleStatus)

	return s
}

func (s *StatusServer) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.server.Shutdown(shutdownCtx)
	}()

	err := s.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *StatusServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	snapshot, err := s.service.Snapshot(r.Context())
	if err != nil {
		if s.logger != nil {
			s.logger.Printf("status snapshot error: %v", err)
		}
		http.Error(w, "status unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(snapshot); err != nil && s.logger != nil {
		s.logger.Printf("status response encode error: %v", err)
	}
}
