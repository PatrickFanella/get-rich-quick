package signal

import (
	"strings"
	"sync"

	"github.com/google/uuid"
)

// StrategyWithThesis carries the minimal strategy context the WatchIndex needs
// to build its term → strategy mapping.
type StrategyWithThesis struct {
	ID         uuid.UUID
	Ticker     string   // auto-added to index
	WatchTerms []string // LLM-generated terms from active thesis
}

// WatchIndex is an in-memory inverted index used as Stage 1 of the two-stage
// signal evaluator. It maps normalised terms to the strategy IDs that care
// about them, enabling zero-latency, zero-cost keyword filtering before any
// LLM call.
//
// The index is built from three sources (union of all):
//  1. Strategy tickers and market slugs (auto-derived)
//  2. WatchTerms from each strategy's active thesis (LLM-generated)
//  3. Manual additions via AddManual / RemoveManual
type WatchIndex struct {
	mu     sync.RWMutex
	auto   map[string][]uuid.UUID // built by Rebuild; cleared on each Rebuild
	manual map[string][]uuid.UUID // persists across Rebuilds
}

// NewWatchIndex returns an empty WatchIndex ready for use.
func NewWatchIndex() *WatchIndex {
	return &WatchIndex{
		auto:   make(map[string][]uuid.UUID),
		manual: make(map[string][]uuid.UUID),
	}
}

// Rebuild replaces the auto-derived index entries from the given strategies.
// Manual entries are preserved. Safe to call concurrently.
func (w *WatchIndex) Rebuild(strategies []StrategyWithThesis) {
	next := make(map[string][]uuid.UUID, len(strategies)*4)
	for _, s := range strategies {
		addTerm(next, s.Ticker, s.ID)
		for _, term := range s.WatchTerms {
			addTerm(next, term, s.ID)
		}
	}

	w.mu.Lock()
	w.auto = next
	w.mu.Unlock()
}

// AddManual registers a manual term → strategyID mapping. Persists across
// Rebuild calls. Safe to call concurrently.
func (w *WatchIndex) AddManual(term string, strategyID uuid.UUID) {
	t := normalise(term)
	if t == "" {
		return
	}
	w.mu.Lock()
	addTerm(w.manual, t, strategyID)
	w.mu.Unlock()
}

// RemoveManual removes a manual term → strategyID mapping. No-op if the
// mapping does not exist. Safe to call concurrently.
func (w *WatchIndex) RemoveManual(term string, strategyID uuid.UUID) {
	t := normalise(term)
	if t == "" {
		return
	}
	w.mu.Lock()
	w.manual[t] = removeID(w.manual[t], strategyID)
	if len(w.manual[t]) == 0 {
		delete(w.manual, t)
	}
	w.mu.Unlock()
}

// Match scans text (case-insensitive) for known terms and returns the
// deduplicated set of strategy IDs that match. Returns nil when no strategies
// match.
func (w *WatchIndex) Match(text string) []uuid.UUID {
	lower := strings.ToLower(text)

	w.mu.RLock()
	defer w.mu.RUnlock()

	seen := make(map[uuid.UUID]struct{})
	for term, ids := range w.auto {
		if strings.Contains(lower, term) {
			for _, id := range ids {
				seen[id] = struct{}{}
			}
		}
	}
	for term, ids := range w.manual {
		if strings.Contains(lower, term) {
			for _, id := range ids {
				seen[id] = struct{}{}
			}
		}
	}

	if len(seen) == 0 {
		return nil
	}
	result := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func normalise(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func addTerm(m map[string][]uuid.UUID, term string, id uuid.UUID) {
	t := normalise(term)
	if t == "" {
		return
	}
	for _, existing := range m[t] {
		if existing == id {
			return // deduplicate
		}
	}
	m[t] = append(m[t], id)
}

func removeID(ids []uuid.UUID, target uuid.UUID) []uuid.UUID {
	out := ids[:0]
	for _, id := range ids {
		if id != target {
			out = append(out, id)
		}
	}
	return out
}
