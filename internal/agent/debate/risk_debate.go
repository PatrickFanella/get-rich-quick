package debate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
)

// RiskDebate is an orchestration Node that runs the full risk debate sequence:
// N rounds of aggressive/conservative/neutral arguments followed by a manager
// (judge) that produces the final risk-adjusted signal.
//
// This node focuses on state orchestration: creating rounds, running debaters,
// and invoking the judge. Pipeline-level concerns such as decision persistence,
// phase timeouts, and DebateRoundCompleted event emission are handled by the
// generic DebateExecutor (internal/agent/debate_executor.go), which the
// Pipeline wires separately.
type RiskDebate struct {
	aggressive   agent.Node
	conservative agent.Node
	neutral      agent.Node
	manager      agent.Node
	rounds       int
	logger       *slog.Logger
}

// NewRiskDebate returns a RiskDebate wired to the given aggressive,
// conservative, neutral, and manager nodes. rounds controls how many debate
// rounds are executed (clamped to a minimum of 1). A nil logger is replaced
// with the default logger.
func NewRiskDebate(aggressive, conservative, neutral, manager agent.Node, rounds int, logger *slog.Logger) *RiskDebate {
	if logger == nil {
		logger = slog.Default()
	}
	if rounds < 1 {
		logger.Warn("debate/risk_debate: invalid rounds; clamping to 1",
			slog.Int("configured_rounds", rounds),
		)
		rounds = 1
	}
	return &RiskDebate{
		aggressive:   aggressive,
		conservative: conservative,
		neutral:      neutral,
		manager:      manager,
		rounds:       rounds,
		logger:       logger,
	}
}

// Name returns the human-readable name for this orchestration node.
func (rd *RiskDebate) Name() string { return "risk_debate" }

// Role returns the agent role for the orchestration node. It delegates to the
// manager (judge) node because the risk debate's final output is the
// risk-adjusted signal produced by the judge. Returns an empty role when the
// manager node is nil.
func (rd *RiskDebate) Role() agent.AgentRole {
	if rd.manager == nil {
		return ""
	}
	return rd.manager.Role()
}

// Phase returns the pipeline phase this node belongs to.
func (rd *RiskDebate) Phase() agent.Phase { return agent.PhaseRiskDebate }

// Execute runs the full risk debate sequence:
//  1. For each round (1..N): append a new DebateRound to the state, then
//     execute the aggressive node, followed by conservative, then neutral.
//  2. Execute the manager node to synthesize all rounds into a final signal.
func (rd *RiskDebate) Execute(ctx context.Context, state *agent.PipelineState) error {
	if rd.aggressive == nil {
		return fmt.Errorf("debate/risk_debate: nil aggressive node")
	}
	if rd.conservative == nil {
		return fmt.Errorf("debate/risk_debate: nil conservative node")
	}
	if rd.neutral == nil {
		return fmt.Errorf("debate/risk_debate: nil neutral node")
	}
	if rd.manager == nil {
		return fmt.Errorf("debate/risk_debate: nil manager node")
	}

	for i := 1; i <= rd.rounds; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		rd.logger.Info("debate/risk_debate: starting round",
			slog.Int("round", i),
			slog.Int("total_rounds", rd.rounds),
		)

		// Append a fresh round so debaters can store contributions.
		state.RiskDebate.Rounds = append(state.RiskDebate.Rounds, agent.DebateRound{
			Number:        i,
			Contributions: make(map[agent.AgentRole]string),
		})

		if err := rd.aggressive.Execute(ctx, state); err != nil {
			return fmt.Errorf("debate/risk_debate: aggressive round %d: %w", i, err)
		}
		if err := rd.conservative.Execute(ctx, state); err != nil {
			return fmt.Errorf("debate/risk_debate: conservative round %d: %w", i, err)
		}
		if err := rd.neutral.Execute(ctx, state); err != nil {
			return fmt.Errorf("debate/risk_debate: neutral round %d: %w", i, err)
		}
	}

	rd.logger.Info("debate/risk_debate: all rounds complete; running manager")

	if err := rd.manager.Execute(ctx, state); err != nil {
		return fmt.Errorf("debate/risk_debate: manager: %w", err)
	}

	return nil
}
