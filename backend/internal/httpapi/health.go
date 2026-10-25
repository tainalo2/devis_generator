package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// Pinger is satisfied by *pgxpool.Pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database,omitempty"`
	Version  string `json:"version"`
}

// handleHealthz is a liveness probe: always 200 if the process can serve HTTP.
// It must not depend on PostgreSQL (so a replica can stay "alive" while DB is warming).
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Version: s.version,
	})
}

// handleReadyz is a readiness probe: 200 only when the database answers Ping.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{
		Status:   "ok",
		Database: "ok",
		Version:  s.version,
	}

	if s.db == nil {
		resp.Status = "not_ready"
		resp.Database = "unconfigured"
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := s.db.Ping(ctx); err != nil {
		resp.Status = "not_ready"
		resp.Database = "unavailable"
		writeJSON(w, http.StatusServiceUnavailable, resp)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
