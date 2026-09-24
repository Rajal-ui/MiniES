package ingest

import (
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rajal-ui/minies/internal/pipeline"
	"github.com/rajal-ui/minies/pkg/logrecord"
)

type Server struct {
	eventLoop *pipeline.EventLoop
	addr      string
}

func NewServer(addr string, eventLoop *pipeline.EventLoop) *Server {
	return &Server{addr: addr, eventLoop: eventLoop}
}

func (s *Server) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var rec logrecord.LogRecord
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if err := rec.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.eventLoop.Ingest(&rec); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func (s *Server) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "missing query parameter q", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":         query,
		"took_ms":       0,
		"total_matches": 0,
		"results":       []interface{}{},
	})
}

func (s *Server) HandleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	stats := s.eventLoop.Stats()
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func RunServer(addr string, eventLoop *pipeline.EventLoop) error {
	s := NewServer(addr, eventLoop)
	http.HandleFunc("/ingest", s.HandleIngest)
	http.HandleFunc("/search", s.HandleSearch)
	http.HandleFunc("/stats", s.HandleStats)
	http.HandleFunc("/health", s.HandleHealth)

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil && err != http.ErrServerClosed {
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	eventLoop.Stop()
	return nil
}
