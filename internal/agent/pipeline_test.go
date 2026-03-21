package agent_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
)

// phaseNode is a minimal Node stub whose Phase() returns a configurable value.
type phaseNode struct {
	name  string
	phase agent.Phase
}

func (n phaseNode) Name() string                                     { return n.name }
func (n phaseNode) Phase() agent.Phase                               { return n.phase }
func (n phaseNode) Execute(context.Context, *agent.PipelineState) error { return nil }

func TestNewPipelineSetsDefaults(t *testing.T) {
	events := make(chan agent.PipelineEvent, 1)
	p := agent.NewPipeline(
		agent.PipelineConfig{}, // zero value — defaults should be applied
		nil,
		nil,
		events,
		slog.Default(),
	)

	cfg := p.Config()
	if cfg.ResearchDebateRounds != 3 {
		t.Errorf("ResearchDebateRounds = %d, want 3", cfg.ResearchDebateRounds)
	}
	if cfg.RiskDebateRounds != 3 {
		t.Errorf("RiskDebateRounds = %d, want 3", cfg.RiskDebateRounds)
	}
}

func TestNewPipelinePreservesExplicitConfig(t *testing.T) {
	events := make(chan agent.PipelineEvent, 1)
	cfg := agent.PipelineConfig{
		PipelineTimeout:      5 * time.Minute,
		PhaseTimeout:         90 * time.Second,
		ResearchDebateRounds: 5,
		RiskDebateRounds:     2,
	}
	p := agent.NewPipeline(cfg, nil, nil, events, slog.Default())

	got := p.Config()
	if got.ResearchDebateRounds != 5 {
		t.Errorf("ResearchDebateRounds = %d, want 5", got.ResearchDebateRounds)
	}
	if got.RiskDebateRounds != 2 {
		t.Errorf("RiskDebateRounds = %d, want 2", got.RiskDebateRounds)
	}
	if got.PipelineTimeout != 5*time.Minute {
		t.Errorf("PipelineTimeout = %v, want 5m", got.PipelineTimeout)
	}
	if got.PhaseTimeout != 90*time.Second {
		t.Errorf("PhaseTimeout = %v, want 90s", got.PhaseTimeout)
	}
}

func TestRegisterNodeGroupsByPhase(t *testing.T) {
	events := make(chan agent.PipelineEvent, 1)
	p := agent.NewPipeline(agent.PipelineConfig{}, nil, nil, events, slog.Default())

	analyst := phaseNode{name: "market_analyst", phase: agent.PhaseAnalysis}
	bull := phaseNode{name: "bull_researcher", phase: agent.PhaseResearchDebate}
	bear := phaseNode{name: "bear_researcher", phase: agent.PhaseResearchDebate}
	trader := phaseNode{name: "trader", phase: agent.PhaseTrading}
	risk := phaseNode{name: "risk_manager", phase: agent.PhaseRiskDebate}

	for _, n := range []agent.Node{analyst, bull, bear, trader, risk} {
		p.RegisterNode(n)
	}

	nodes := p.Nodes()

	checkPhase := func(phase agent.Phase, wantNames []string) {
		t.Helper()
		got := nodes[phase]
		if len(got) != len(wantNames) {
			t.Errorf("phase %q: got %d nodes, want %d", phase, len(got), len(wantNames))
			return
		}
		for i, n := range got {
			if n.Name() != wantNames[i] {
				t.Errorf("phase %q node[%d].Name() = %q, want %q", phase, i, n.Name(), wantNames[i])
			}
		}
	}

	checkPhase(agent.PhaseAnalysis, []string{"market_analyst"})
	checkPhase(agent.PhaseResearchDebate, []string{"bull_researcher", "bear_researcher"})
	checkPhase(agent.PhaseTrading, []string{"trader"})
	checkPhase(agent.PhaseRiskDebate, []string{"risk_manager"})
}
