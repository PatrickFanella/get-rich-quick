package api

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"

	pgrepo "github.com/PatrickFanella/get-rich-quick/internal/repository/postgres"
)

// ReportMetrics captures report staleness observations.
type ReportMetrics interface {
	ObserveReportStaleness(strategyID string, seconds float64)
}

// ReportArtifactStore captures report artifact reads used by report handlers.
type ReportArtifactStore interface {
	GetLatest(ctx context.Context, strategyID uuid.UUID, reportType string) (*pgrepo.ReportArtifact, error)
	List(ctx context.Context, filter pgrepo.ReportArtifactFilter, limit, offset int) ([]pgrepo.ReportArtifact, error)
}

// reportLatestResponse wraps the latest report artifact with a stale_seconds
// field showing how old the report is.
type reportLatestResponse struct {
	pgrepo.ReportArtifact
	StaleSeconds float64 `json:"stale_seconds"`
}

// handleGetLatestReport returns the most recently completed report artifact
// for a given strategy.
//
//	GET /api/v1/strategies/{id}/reports/latest
func (s *Server) handleGetLatestReport(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), ErrCodeBadRequest)
		return
	}
	if s.reportArtifacts == nil {
		respondError(w, http.StatusNotImplemented, "report artifacts not configured", ErrCodeNotImplemented)
		return
	}

	reportType := r.URL.Query().Get("report_type")
	if reportType == "" {
		reportType = "paper_validation"
	}

	artifact, err := s.reportArtifacts.GetLatest(r.Context(), id, reportType)
	if err != nil {
		if isNotFound(err) {
			respondError(w, http.StatusNotFound, "no completed report found", ErrCodeNotFound)
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get latest report", ErrCodeInternal)
		return
	}

	stale := 0.0
	if artifact.CompletedAt != nil {
		stale = math.Max(0, math.Round(time.Since(*artifact.CompletedAt).Seconds()))
	}

	if s.reportMetrics != nil {
		s.reportMetrics.ObserveReportStaleness(id.String(), stale)
	}

	respondJSON(w, http.StatusOK, reportLatestResponse{
		ReportArtifact: *artifact,
		StaleSeconds:   stale,
	})
}

// handleListReports returns a paginated list of report artifacts for a strategy.
//
//	GET /api/v1/strategies/{id}/reports
func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error(), ErrCodeBadRequest)
		return
	}
	if s.reportArtifacts == nil {
		respondError(w, http.StatusNotImplemented, "report artifacts not configured", ErrCodeNotImplemented)
		return
	}

	limit, offset := parsePagination(r)

	filter := pgrepo.ReportArtifactFilter{
		StrategyID: &id,
	}
	if rt := r.URL.Query().Get("report_type"); rt != "" {
		filter.ReportType = rt
	}
	if st := r.URL.Query().Get("status"); st != "" {
		filter.Status = st
	}

	artifacts, err := s.reportArtifacts.List(r.Context(), filter, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list reports", ErrCodeInternal)
		return
	}

	respondList(w, artifacts, limit, offset)
}
