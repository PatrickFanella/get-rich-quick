package signal

import "github.com/google/uuid"

// EvaluatedSignal is a RawSignalEvent that has passed the keyword filter
// and been scored by the LLM evaluator.
type EvaluatedSignal struct {
	Raw                RawSignalEvent
	AffectedStrategies []uuid.UUID
	Urgency            int    // 1–5; 1=noise, 5=critical/breaking
	Summary            string // one-line LLM summary
	RecommendedAction  string // "monitor", "re-evaluate", or "execute_thesis"
}
