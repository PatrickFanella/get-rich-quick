package domain

import (
	"time"

	"github.com/google/uuid"
)

// AgentMemory represents a learned recommendation stored by an agent for future retrieval.
type AgentMemory struct {
	ID             uuid.UUID  `json:"id"`
	AgentRole      AgentRole  `json:"agent_role"`
	Situation      string     `json:"situation"`
	Recommendation string     `json:"recommendation"`
	Outcome        string     `json:"outcome,omitempty"`
	PipelineRunID  *uuid.UUID `json:"pipeline_run_id,omitempty"`
	RelevanceScore *float64   `json:"relevance_score,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}
