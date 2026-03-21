package agent

import (
	"log/slog"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/repository"
)

// PipelineConfig holds timeout and debate-round configuration for a Pipeline.
type PipelineConfig struct {
	PipelineTimeout      time.Duration
	PhaseTimeout         time.Duration
	ResearchDebateRounds int
	RiskDebateRounds     int
}

// Pipeline holds all dependencies and configuration needed by the executor.
type Pipeline struct {
	nodes             map[Phase][]Node
	pipelineRunRepo   repository.PipelineRunRepository
	agentDecisionRepo repository.AgentDecisionRepository
	events            chan<- PipelineEvent
	logger            *slog.Logger
	config            PipelineConfig
}

// NewPipeline constructs a Pipeline with the supplied dependencies. Default
// debate-round counts of 3 are applied when the config fields are zero.
func NewPipeline(
	config PipelineConfig,
	pipelineRunRepo repository.PipelineRunRepository,
	agentDecisionRepo repository.AgentDecisionRepository,
	events chan<- PipelineEvent,
	logger *slog.Logger,
) *Pipeline {
	if config.ResearchDebateRounds == 0 {
		config.ResearchDebateRounds = 3
	}
	if config.RiskDebateRounds == 0 {
		config.RiskDebateRounds = 3
	}
	return &Pipeline{
		nodes:             make(map[Phase][]Node),
		pipelineRunRepo:   pipelineRunRepo,
		agentDecisionRepo: agentDecisionRepo,
		events:            events,
		logger:            logger,
		config:            config,
	}
}

// RegisterNode adds a node to the phase group determined by node.Phase().
func (p *Pipeline) RegisterNode(node Node) {
	phase := node.Phase()
	p.nodes[phase] = append(p.nodes[phase], node)
}

// Config returns the resolved PipelineConfig (with defaults applied).
func (p *Pipeline) Config() PipelineConfig {
	return p.config
}

// Nodes returns a copy of the phase-to-nodes map for inspection.
func (p *Pipeline) Nodes() map[Phase][]Node {
	out := make(map[Phase][]Node, len(p.nodes))
	for phase, nodes := range p.nodes {
		out[phase] = append([]Node(nil), nodes...)
	}
	return out
}
