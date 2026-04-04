package universe

import (
	"context"
	"time"
)

// TrackedTicker represents a ticker tracked in the universe.
type TrackedTicker struct {
	Ticker      string     `json:"ticker"`
	Name        string     `json:"name"`
	Exchange    string     `json:"exchange"`
	IndexGroup  string     `json:"index_group"`
	WatchScore  float64    `json:"watch_score"`
	LastScanned *time.Time `json:"last_scanned,omitempty"`
	Active      bool       `json:"active"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// UniverseRepository provides storage operations for the ticker universe.
type UniverseRepository interface {
	Upsert(ctx context.Context, ticker *TrackedTicker) error
	UpsertBatch(ctx context.Context, tickers []TrackedTicker) error
	List(ctx context.Context, filter ListFilter, limit, offset int) ([]TrackedTicker, error)
	Watchlist(ctx context.Context, topN int) ([]TrackedTicker, error)
	UpdateScore(ctx context.Context, ticker string, score float64) error
	Count(ctx context.Context) (int, error)
}

// ListFilter defines supported filters when listing universe tickers.
type ListFilter struct {
	IndexGroup string
	Active     *bool
	Search     string // ticker or name ILIKE
}
