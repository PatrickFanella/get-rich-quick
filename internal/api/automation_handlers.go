package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// handleGetAutomationStatus returns status for all registered jobs.
// GET /api/v1/automation/status
func (s *Server) handleGetAutomationStatus(w http.ResponseWriter, r *http.Request) {
	if s.automation == nil {
		respondError(w, http.StatusServiceUnavailable, "automation not configured", ErrCodeInternal)
		return
	}
	respondJSON(w, http.StatusOK, s.automation.Status())
}

// handleRunAutomationJob triggers a specific job by name.
// POST /api/v1/automation/jobs/{name}/run
func (s *Server) handleRunAutomationJob(w http.ResponseWriter, r *http.Request) {
	if s.automation == nil {
		respondError(w, http.StatusServiceUnavailable, "automation not configured", ErrCodeInternal)
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "job name is required", ErrCodeBadRequest)
		return
	}

	if err := s.automation.RunJob(r.Context(), name); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), ErrCodeBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "triggered"})
}

// handleSetAutomationJobEnabled enables or disables a job.
// POST /api/v1/automation/jobs/{name}/enable
// Body: {"enabled": true}
func (s *Server) handleSetAutomationJobEnabled(w http.ResponseWriter, r *http.Request) {
	if s.automation == nil {
		respondError(w, http.StatusServiceUnavailable, "automation not configured", ErrCodeInternal)
		return
	}

	name := chi.URLParam(r, "name")
	if name == "" {
		respondError(w, http.StatusBadRequest, "job name is required", ErrCodeBadRequest)
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", ErrCodeBadRequest)
		return
	}

	if err := s.automation.SetEnabled(name, req.Enabled); err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), ErrCodeBadRequest)
		return
	}

	respondJSON(w, http.StatusOK, map[string]bool{"enabled": req.Enabled})
}
