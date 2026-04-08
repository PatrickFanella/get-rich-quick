package signal

import "github.com/google/uuid"

// TriggerAction describes the dispatch action to take for a matched signal.
type TriggerAction string

const (
	TriggerActionLogOnly       TriggerAction = "log_only"
	TriggerActionRunPipeline   TriggerAction = "run_pipeline"
	TriggerActionExecuteThesis TriggerAction = "execute_thesis"
)

// TriggerEvent is produced by the urgency-tiered dispatcher and consumed
// by the automation orchestrator.
type TriggerEvent struct {
	Signal     EvaluatedSignal
	StrategyID uuid.UUID
	Action     TriggerAction
	Priority   int // derived from Signal.Urgency
}
