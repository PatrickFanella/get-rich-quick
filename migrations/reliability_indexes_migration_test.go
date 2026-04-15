package migrations_test

import (
	"strings"
	"testing"
)

func TestReliabilityIndexesUpMigrationDefinesExpectedIndexesWithoutConcurrently(t *testing.T) {
	upSQL := normalizeSQL(t, readMigrationFile(t, "000027_reliability_indexes.up.sql"))

	for _, fragment := range []string{
		"create index if not exists idx_pipeline_runs_status_started on pipeline_runs(status, started_at) where status = 'running'",
		"create index if not exists idx_automation_job_runs_job_name_started on automation_job_runs(job_name, started_at desc)",
		"create index if not exists idx_agent_decisions_run_id on agent_decisions(pipeline_run_id)",
	} {
		if !strings.Contains(upSQL, fragment) {
			t.Fatalf("expected up migration to contain %q, got:\n%s", fragment, upSQL)
		}
	}

	if strings.Contains(upSQL, "concurrently") {
		t.Fatalf("expected up migration to avoid concurrently so it remains tx-safe, got:\n%s", upSQL)
	}
}
