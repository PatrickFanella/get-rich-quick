package postgres

import (
	"testing"
)

func TestQueryBuilderAddCondition(t *testing.T) {
	qb := NewQueryBuilder()
	qb.AddCondition("ticker", "=", "AAPL")
	qb.AddCondition("status", "=", "open")

	got := qb.WhereClause()
	want := " WHERE ticker = $1 AND status = $2"
	if got != want {
		t.Fatalf("WhereClause() = %q, want %q", got, want)
	}
	if len(qb.Args()) != 2 {
		t.Fatalf("Args() len = %d, want 2", len(qb.Args()))
	}
	if qb.Args()[0] != "AAPL" {
		t.Fatalf("Args()[0] = %v, want AAPL", qb.Args()[0])
	}
}

func TestQueryBuilderWhereClauseEmpty(t *testing.T) {
	qb := NewQueryBuilder()
	if got := qb.WhereClause(); got != "" {
		t.Fatalf("WhereClause() = %q, want empty", got)
	}
}

func TestQueryBuilderAddRawCondition(t *testing.T) {
	qb := NewQueryBuilder()
	qb.AddRawCondition("closed_at IS NULL")
	qb.AddCondition("ticker", "=", "BTC")

	got := qb.WhereClause()
	want := " WHERE closed_at IS NULL AND ticker = $1"
	if got != want {
		t.Fatalf("WhereClause() = %q, want %q", got, want)
	}
}

func TestQueryBuilderPagination(t *testing.T) {
	qb := NewQueryBuilder()
	qb.AddCondition("status", "=", "active")
	query := "SELECT * FROM strategies" + qb.WhereClause()
	query = qb.Pagination(query, 10, 20)

	want := "SELECT * FROM strategies WHERE status = $1 LIMIT $2 OFFSET $3"
	if query != want {
		t.Fatalf("query = %q, want %q", query, want)
	}
	if len(qb.Args()) != 3 {
		t.Fatalf("Args() len = %d, want 3", len(qb.Args()))
	}
}

func TestQueryBuilderNextParam(t *testing.T) {
	qb := NewQueryBuilder()
	p1 := qb.NextParam(42)
	p2 := qb.NextParam("hello")

	if p1 != "$1" {
		t.Fatalf("NextParam(42) = %q, want $1", p1)
	}
	if p2 != "$2" {
		t.Fatalf("NextParam(hello) = %q, want $2", p2)
	}
}

func TestQueryBuilderMixedUsage(t *testing.T) {
	qb := NewQueryBuilder()
	qb.AddCondition("agent_role", "=", "market_analyst")
	qb.AddRawCondition("created_at > NOW() - INTERVAL '30 days'")
	qb.AddCondition("relevance_score", ">=", 0.5)

	got := qb.WhereClause()
	want := " WHERE agent_role = $1 AND created_at > NOW() - INTERVAL '30 days' AND relevance_score >= $2"
	if got != want {
		t.Fatalf("WhereClause() = %q, want %q", got, want)
	}
	if len(qb.Args()) != 2 {
		t.Fatalf("Args() len = %d, want 2", len(qb.Args()))
	}
}
