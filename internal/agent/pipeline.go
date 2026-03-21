package agent

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"
)

// Pipeline orchestrates the execution of registered agent nodes across phases.
type Pipeline struct {
	nodes        []Node
	events       chan<- PipelineEvent
	phaseTimeout time.Duration
	logger       *slog.Logger
}

// NewPipeline constructs a Pipeline with the given nodes, event channel, phase timeout,
// and logger. If logger is nil, slog.Default() is used. If events is nil, no events are
// emitted. If phaseTimeout is zero, no timeout is applied to individual phases.
func NewPipeline(nodes []Node, events chan<- PipelineEvent, phaseTimeout time.Duration, logger *slog.Logger) *Pipeline {
	if logger == nil {
		logger = slog.Default()
	}
	return &Pipeline{
		nodes:        nodes,
		events:       events,
		phaseTimeout: phaseTimeout,
		logger:       logger,
	}
}

// executeAnalysisPhase runs all registered PhaseAnalysis nodes concurrently using
// errgroup. If any node fails, a warning is logged and the remaining nodes continue
// unaffected (partial failures do not abort the phase). If phaseTimeout is positive, it
// is applied as a deadline for the entire phase, cancelling any nodes that have not yet
// completed. An AgentDecisionMade event is emitted after each node completes successfully.
//
// This method always returns nil; analyst node failures are tolerated and surfaced only
// through log warnings. The error return is reserved for future structural failures
// (e.g., a cancelled parent context passed before any node is launched).
func (p *Pipeline) executeAnalysisPhase(ctx context.Context, state *PipelineState) error {
	phaseCtx := ctx
	if p.phaseTimeout > 0 {
		var cancel context.CancelFunc
		phaseCtx, cancel = context.WithTimeout(ctx, p.phaseTimeout)
		defer cancel()
	}

	g, gCtx := errgroup.WithContext(phaseCtx)

	for _, n := range p.nodes {
		if n.Phase() != PhaseAnalysis {
			continue
		}
		node := n
		g.Go(func() error {
			if err := node.Execute(gCtx, state); err != nil {
				p.logger.Warn("agent/pipeline: analyst node failed",
					slog.String("node", node.Name()),
					slog.Any("error", err),
				)
				return nil // partial failures are tolerated; do not abort the phase
			}

			if p.events != nil {
				p.events <- PipelineEvent{
					Type:          AgentDecisionMade,
					PipelineRunID: state.PipelineRunID,
					StrategyID:    state.StrategyID,
					Ticker:        state.Ticker,
					AgentRole:     AgentRole(node.Name()),
					Phase:         PhaseAnalysis,
					OccurredAt:    time.Now().UTC(),
				}
			}
			return nil
		})
	}

	return g.Wait()
}
