package agent

import "context"

// Node represents a single executable step in the agent pipeline.
type Node interface {
	Name() string
	Phase() Phase
	Execute(ctx context.Context, state *PipelineState) error
}
