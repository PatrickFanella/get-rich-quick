package agent_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
)

type stubNode struct{}

func (stubNode) Name() string { return "stub" }

func (stubNode) Phase() agent.Phase { return agent.PhaseAnalysis }

func (stubNode) Execute(context.Context, *agent.PipelineState) error { return nil }

func TestNodeInterface(t *testing.T) {
	var _ agent.Node = stubNode{}
}

func TestPipelineStateStoresPhaseOutputs(t *testing.T) {
	runID := uuid.New()
	strategyID := uuid.New()
	expectedErr := errors.New("risk rejected trade")

	state := agent.PipelineState{
		RunID:      runID,
		StrategyID: strategyID,
		Ticker:     "AAPL",
		AnalystReports: map[agent.AgentRole]string{
			agent.AgentRoleMarketAnalyst:  "Uptrend confirmed",
			agent.AgentRoleBullResearcher: "Momentum remains strong",
			agent.AgentRoleBearResearcher: "Valuation is stretched",
		},
		ResearchDebate: agent.ResearchDebateState{
			Rounds: []agent.DebateRound{
				{
					Number: 1,
					Contributions: map[agent.AgentRole]string{
						agent.AgentRoleBullResearcher: "Bull case",
						agent.AgentRoleBearResearcher: "Bear case",
					},
				},
			},
			InvestmentPlan: "Accumulate on pullbacks with a defined stop.",
		},
		TradingPlan: agent.TradingPlan{
			Action:       agent.PipelineSignalBuy,
			Ticker:       "AAPL",
			EntryType:    "limit",
			EntryPrice:   215.25,
			PositionSize: 5000,
			StopLoss:     209.50,
			TakeProfit:   228.00,
			TimeHorizon:  "swing",
			Confidence:   0.78,
			Rationale:    "Momentum and analyst alignment support a swing entry.",
			RiskReward:   2.2,
		},
		RiskDebate: agent.RiskDebateState{
			Rounds: []agent.DebateRound{
				{
					Number: 1,
					Contributions: map[agent.AgentRole]string{
						agent.AgentRoleRiskManager: "Size should remain below normal due to earnings proximity.",
					},
				},
			},
			FinalSignal: "Approve reduced-size long position.",
		},
		FinalSignal: agent.FinalSignal{
			Signal:     agent.PipelineSignalBuy,
			Confidence: 0.72,
		},
		Errors: []error{expectedErr},
	}

	if state.RunID != runID {
		t.Fatalf("RunID = %v, want %v", state.RunID, runID)
	}
	if state.StrategyID != strategyID {
		t.Fatalf("StrategyID = %v, want %v", state.StrategyID, strategyID)
	}
	if got := state.AnalystReports[agent.AgentRoleMarketAnalyst]; got != "Uptrend confirmed" {
		t.Fatalf("AnalystReports[market_analyst] = %q, want %q", got, "Uptrend confirmed")
	}
	if got := state.ResearchDebate.InvestmentPlan; got != "Accumulate on pullbacks with a defined stop." {
		t.Fatalf("ResearchDebate.InvestmentPlan = %q, want expected plan", got)
	}
	if got := state.TradingPlan.Action; got != agent.PipelineSignalBuy {
		t.Fatalf("TradingPlan.Action = %q, want %q", got, agent.PipelineSignalBuy)
	}
	if got := state.RiskDebate.FinalSignal; got != "Approve reduced-size long position." {
		t.Fatalf("RiskDebate.FinalSignal = %q, want expected verdict", got)
	}
	if got := state.FinalSignal.Signal; got != agent.PipelineSignalBuy {
		t.Fatalf("FinalSignal.Signal = %q, want %q", got, agent.PipelineSignalBuy)
	}
	if len(state.Errors) != 1 || !errors.Is(state.Errors[0], expectedErr) {
		t.Fatalf("Errors = %v, want [%v]", state.Errors, expectedErr)
	}
}

func TestPipelineEventTypesCoverUserVisibleTransitions(t *testing.T) {
	timestamp := time.Now().UTC()

	event := agent.PipelineEvent{
		Type:       agent.SignalGenerated,
		RunID:      uuid.New(),
		StrategyID: uuid.New(),
		Ticker:     "AAPL",
		AgentRole:  agent.AgentRoleRiskManager,
		Phase:      agent.PhaseRiskDebate,
		Round:      2,
		Payload: agent.FinalSignal{
			Signal:     agent.PipelineSignalHold,
			Confidence: 0.64,
		},
		OccurredAt: timestamp,
	}

	if got := agent.PipelineStarted.String(); got != "pipeline_started" {
		t.Fatalf("PipelineStarted.String() = %q, want %q", got, "pipeline_started")
	}
	if got := agent.AgentDecisionMade.String(); got != "agent_decision_made" {
		t.Fatalf("AgentDecisionMade.String() = %q, want %q", got, "agent_decision_made")
	}
	if got := agent.DebateRoundCompleted.String(); got != "debate_round_completed" {
		t.Fatalf("DebateRoundCompleted.String() = %q, want %q", got, "debate_round_completed")
	}
	if got := agent.SignalGenerated.String(); got != "signal_generated" {
		t.Fatalf("SignalGenerated.String() = %q, want %q", got, "signal_generated")
	}
	if got := agent.PipelineCompleted.String(); got != "pipeline_completed" {
		t.Fatalf("PipelineCompleted.String() = %q, want %q", got, "pipeline_completed")
	}
	if got := agent.PipelineError.String(); got != "pipeline_error" {
		t.Fatalf("PipelineError.String() = %q, want %q", got, "pipeline_error")
	}

	payload, ok := event.Payload.(agent.FinalSignal)
	if !ok {
		t.Fatalf("Payload type = %T, want agent.FinalSignal", event.Payload)
	}
	if payload.Signal != agent.PipelineSignalHold || payload.Confidence != 0.64 {
		t.Fatalf("Payload = %+v, want hold @ 0.64", payload)
	}
}
