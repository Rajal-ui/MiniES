package ingest

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/rajal-ui/minies/internal/pipeline"
	"github.com/rajal-ui/minies/internal/search"
	"github.com/rajal-ui/minies/pkg/logrecord"
)

type Server struct {
	eventLoop *pipeline.EventLoop
	executor  *search.Executor
	addr      string
}

func NewServer(addr string, eventLoop *pipeline.EventLoop, executor *search.Executor) *Server {
	return &Server{addr: addr, eventLoop: eventLoop, executor: executor}
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
	start := time.Now()
	results, err := s.executor.Search(query)
	tookMs := time.Since(start).Milliseconds()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"error":   err.Error(),
			"results": []interface{}{},
		})
		return
	}
	if results == nil {
		results = []search.SearchResult{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query":         query,
		"took_ms":       tookMs,
		"total_matches": len(results),
		"results":       results,
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

func RunServer(addr string, eventLoop *pipeline.EventLoop, executor *search.Executor) error {
	s := NewServer(addr, eventLoop, executor)
	http.HandleFunc("/ingest", s.HandleIngest)
	http.HandleFunc("/search", s.HandleSearch)
	http.HandleFunc("/stats", s.HandleStats)
	http.HandleFunc("/health", s.HandleHealth)

	return http.ListenAndServe(addr, nil)
}
