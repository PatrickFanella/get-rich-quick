package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"

	pgrepo "github.com/PatrickFanella/get-rich-quick/internal/repository/postgres"
)

// These tests exercise the "not configured" handler path by using the
// default test server setup, where Server.reportArtifacts is left nil.

func TestHandleGetLatestReport_NotConfigured(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	// reportArtifacts is nil by default → 501
	rr := doRequest(t, srv, http.MethodGet, "/api/v1/strategies/"+stratA.ID.String()+"/reports/latest", nil)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
	}
}

func TestHandleListReports_NotConfigured(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	rr := doRequest(t, srv, http.MethodGet, "/api/v1/strategies/"+stratA.ID.String()+"/reports", nil)
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotImplemented)
	}
}

func TestHandleGetLatestReport_InvalidUUID(t *testing.T) {
	t.Parallel()

	srv := newTestServer(t)
	rr := doRequest(t, srv, http.MethodGet, "/api/v1/strategies/not-a-uuid/reports/latest", nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestReportLatestResponse_StaleSeconds(t *testing.T) {
	t.Parallel()

	completed := time.Now().Add(-5 * time.Minute)
	resp := reportLatestResponse{
		ReportArtifact: pgrepo.ReportArtifact{
			ID:          uuid.New(),
			StrategyID:  stratA.ID,
			ReportType:  "paper_validation",
			TimeBucket:  time.Now().Truncate(24 * time.Hour),
			Status:      "completed",
			ReportJSON:  json.RawMessage(`{"decision":"GO"}`),
			CompletedAt: &completed,
		},
		StaleSeconds: 300,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	stale, ok := got["stale_seconds"].(float64)
	if !ok {
		t.Fatal("stale_seconds not present in response")
	}
	if stale != 300 {
		t.Fatalf("stale_seconds = %f, want 300", stale)
	}
}
