package server

import (
	"fmt"
	"net/http"

	"github.com/andymarkow/go-metrics-collector/internal/handlers"
	"github.com/andymarkow/go-metrics-collector/internal/storage"
)

type Server struct {
	addr string
	mux  *http.ServeMux
}

func NewServer() *Server {
	mStorage := storage.NewMemStorage()

	h := handlers.NewHandlers(mStorage)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /update/{metricType}/{metricName}/{metricValue}", h.UpdateMetric)

	return &Server{
		addr: "0.0.0.0:8080",
		mux:  mux,
	}
}

func (s *Server) Start() error {
	if err := http.ListenAndServe(s.addr, s.mux); err != nil {
		return fmt.Errorf("http.ListenAndServe: %w", err)
	}

	return nil
}
