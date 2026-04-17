package postgres_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	pgrepo "github.com/PatrickFanella/get-rich-quick/internal/repository/postgres"
)

func TestReportArtifact_RoundTrip(t *testing.T) {
	// Unit-level: verify struct serialisation and field assignment.
	now := time.Now().UTC().Truncate(time.Second)
	report := json.RawMessage(`{"decision":"GO"}`)
	completed := now

	a := &pgrepo.ReportArtifact{
		ID:               uuid.New(),
		StrategyID:       uuid.New(),
		ReportType:       "paper_validation",
		TimeBucket:       now.Truncate(24 * time.Hour),
		Status:           "completed",
		ReportJSON:       report,
		Provider:         "openrouter",
		Model:            "meta-llama/llama-3.3-70b-instruct:free",
		PromptTokens:     100,
		CompletionTokens: 50,
		LatencyMs:        1200,
		CreatedAt:        now,
		CompletedAt:      &completed,
	}

	// Verify JSON round-trip.
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got pgrepo.ReportArtifact
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.StrategyID != a.StrategyID {
		t.Errorf("strategy_id = %s, want %s", got.StrategyID, a.StrategyID)
	}
	if got.ReportType != "paper_validation" {
		t.Errorf("report_type = %q, want paper_validation", got.ReportType)
	}
	if got.Status != "completed" {
		t.Errorf("status = %q, want completed", got.Status)
	}
	if got.PromptTokens != 100 {
		t.Errorf("prompt_tokens = %d, want 100", got.PromptTokens)
	}
	if got.CompletionTokens != 50 {
		t.Errorf("completion_tokens = %d, want 50", got.CompletionTokens)
	}
}

func TestReportArtifactFilter_Defaults(t *testing.T) {
	f := pgrepo.ReportArtifactFilter{}
	if f.StrategyID != nil {
		t.Error("expected nil StrategyID")
	}
	if f.ReportType != "" {
		t.Error("expected empty ReportType")
	}
	if f.Status != "" {
		t.Error("expected empty Status")
	}
}
