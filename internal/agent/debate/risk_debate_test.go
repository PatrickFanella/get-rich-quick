package debate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
)

// ---------------------------------------------------------------------------
// Risk-debate-specific stub helpers
// ---------------------------------------------------------------------------

// newAggressiveStub returns a stub that writes a contribution into the current
// risk debate round.
func newAggressiveStub(callLog *[]string) *stubNode {
	return &stubNode{
		name:    "aggressive_risk",
		role:    agent.AgentRoleAggressiveRisk,
		phase:   agent.PhaseRiskDebate,
		callLog: callLog,
		execFn: func(_ context.Context, state *agent.PipelineState) error {
			rounds := state.RiskDebate.Rounds
			if len(rounds) == 0 {
				return nil
			}
			cur := &state.RiskDebate.Rounds[len(rounds)-1]
			if cur.Contributions == nil {
				cur.Contributions = make(map[agent.AgentRole]string)
			}
			cur.Contributions[agent.AgentRoleAggressiveRisk] = fmt.Sprintf("aggressive_r%d", cur.Number)
			return nil
		},
	}
}

// newConservativeStub returns a stub that writes a contribution into the
// current risk debate round.
func newConservativeStub(callLog *[]string) *stubNode {
	return &stubNode{
		name:    "conservative_risk",
		role:    agent.AgentRoleConservativeRisk,
		phase:   agent.PhaseRiskDebate,
		callLog: callLog,
		execFn: func(_ context.Context, state *agent.PipelineState) error {
			rounds := state.RiskDebate.Rounds
			if len(rounds) == 0 {
				return nil
			}
			cur := &state.RiskDebate.Rounds[len(rounds)-1]
			if cur.Contributions == nil {
				cur.Contributions = make(map[agent.AgentRole]string)
			}
			cur.Contributions[agent.AgentRoleConservativeRisk] = fmt.Sprintf("conservative_r%d", cur.Number)
			return nil
		},
	}
}

// newNeutralStub returns a stub that writes a contribution into the current
// risk debate round.
func newNeutralStub(callLog *[]string) *stubNode {
	return &stubNode{
		name:    "neutral_risk",
		role:    agent.AgentRoleNeutralRisk,
		phase:   agent.PhaseRiskDebate,
		callLog: callLog,
		execFn: func(_ context.Context, state *agent.PipelineState) error {
			rounds := state.RiskDebate.Rounds
			if len(rounds) == 0 {
				return nil
			}
			cur := &state.RiskDebate.Rounds[len(rounds)-1]
			if cur.Contributions == nil {
				cur.Contributions = make(map[agent.AgentRole]string)
			}
			cur.Contributions[agent.AgentRoleNeutralRisk] = fmt.Sprintf("neutral_r%d", cur.Number)
			return nil
		},
	}
}

// newRiskManagerStub returns a stub that sets the FinalSignal on the risk
// debate state.
func newRiskManagerStub(callLog *[]string) *stubNode {
	return &stubNode{
		name:    "risk_manager",
		role:    agent.AgentRoleRiskManager,
		phase:   agent.PhaseRiskDebate,
		callLog: callLog,
		execFn: func(_ context.Context, state *agent.PipelineState) error {
			state.RiskDebate.FinalSignal = `{"action":"BUY","confidence":8}`
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestRiskDebateNodeInterface(t *testing.T) {
	callLog := []string{}
	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		newRiskManagerStub(&callLog),
		3,
		slog.Default(),
	)

	if got := rd.Name(); got != "risk_debate" {
		t.Fatalf("Name() = %q, want %q", got, "risk_debate")
	}
	if got := rd.Role(); got != agent.AgentRoleRiskManager {
		t.Fatalf("Role() = %q, want %q", got, agent.AgentRoleRiskManager)
	}
	if got := rd.Phase(); got != agent.PhaseRiskDebate {
		t.Fatalf("Phase() = %q, want %q", got, agent.PhaseRiskDebate)
	}
}

// TestRiskDebateThreeRoundsOrder verifies that 3 rounds execute aggressive,
// then conservative, then neutral in sequence, followed by the manager at the
// end.
func TestRiskDebateThreeRoundsOrder(t *testing.T) {
	callLog := []string{}
	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		newRiskManagerStub(&callLog),
		3,
		slog.Default(),
	)

	state := &agent.PipelineState{}

	if err := rd.Execute(context.Background(), state); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	wantOrder := []string{
		"aggressive_risk", "conservative_risk", "neutral_risk", // round 1
		"aggressive_risk", "conservative_risk", "neutral_risk", // round 2
		"aggressive_risk", "conservative_risk", "neutral_risk", // round 3
		"risk_manager", // judge
	}

	if len(callLog) != len(wantOrder) {
		t.Fatalf("call count = %d, want %d; log = %v", len(callLog), len(wantOrder), callLog)
	}
	for i, want := range wantOrder {
		if callLog[i] != want {
			t.Fatalf("call[%d] = %q, want %q; full log = %v", i, callLog[i], want, callLog)
		}
	}
}

// TestRiskDebateFinalSignalExtracted verifies that the manager node's output
// is stored in state.RiskDebate.FinalSignal after execution.
func TestRiskDebateFinalSignalExtracted(t *testing.T) {
	callLog := []string{}
	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		newRiskManagerStub(&callLog),
		2,
		slog.Default(),
	)

	state := &agent.PipelineState{}

	if err := rd.Execute(context.Background(), state); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	wantSignal := `{"action":"BUY","confidence":8}`
	if got := state.RiskDebate.FinalSignal; got != wantSignal {
		t.Fatalf("FinalSignal = %q, want %q", got, wantSignal)
	}
}

// TestRiskDebateStateAccumulation verifies that rounds and contributions are
// correctly accumulated in PipelineState after a 3-round debate.
func TestRiskDebateStateAccumulation(t *testing.T) {
	callLog := []string{}
	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		newRiskManagerStub(&callLog),
		3,
		slog.Default(),
	)

	state := &agent.PipelineState{}

	if err := rd.Execute(context.Background(), state); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	// Verify 3 rounds were accumulated.
	if got := len(state.RiskDebate.Rounds); got != 3 {
		t.Fatalf("rounds count = %d, want 3", got)
	}

	for i, round := range state.RiskDebate.Rounds {
		wantNum := i + 1
		if round.Number != wantNum {
			t.Fatalf("round[%d].Number = %d, want %d", i, round.Number, wantNum)
		}

		wantAgg := fmt.Sprintf("aggressive_r%d", wantNum)
		if got := round.Contributions[agent.AgentRoleAggressiveRisk]; got != wantAgg {
			t.Fatalf("round[%d] aggressive = %q, want %q", i, got, wantAgg)
		}

		wantCons := fmt.Sprintf("conservative_r%d", wantNum)
		if got := round.Contributions[agent.AgentRoleConservativeRisk]; got != wantCons {
			t.Fatalf("round[%d] conservative = %q, want %q", i, got, wantCons)
		}

		wantNeut := fmt.Sprintf("neutral_r%d", wantNum)
		if got := round.Contributions[agent.AgentRoleNeutralRisk]; got != wantNeut {
			t.Fatalf("round[%d] neutral = %q, want %q", i, got, wantNeut)
		}
	}

	// Verify final signal was set.
	if state.RiskDebate.FinalSignal == "" {
		t.Fatal("FinalSignal is empty, want non-empty")
	}
}

// TestRiskDebateSingleRound verifies that a single-round debate works
// correctly (rounds=1).
func TestRiskDebateSingleRound(t *testing.T) {
	callLog := []string{}
	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		newRiskManagerStub(&callLog),
		1,
		slog.Default(),
	)

	state := &agent.PipelineState{}

	if err := rd.Execute(context.Background(), state); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	wantOrder := []string{"aggressive_risk", "conservative_risk", "neutral_risk", "risk_manager"}
	if len(callLog) != len(wantOrder) {
		t.Fatalf("call count = %d, want %d; log = %v", len(callLog), len(wantOrder), callLog)
	}
	for i, want := range wantOrder {
		if callLog[i] != want {
			t.Fatalf("call[%d] = %q, want %q; full log = %v", i, callLog[i], want, callLog)
		}
	}

	if got := len(state.RiskDebate.Rounds); got != 1 {
		t.Fatalf("rounds count = %d, want 1", got)
	}

	round := state.RiskDebate.Rounds[0]
	if round.Number != 1 {
		t.Fatalf("round.Number = %d, want 1", round.Number)
	}
	if got := round.Contributions[agent.AgentRoleAggressiveRisk]; got != "aggressive_r1" {
		t.Fatalf("aggressive contribution = %q, want %q", got, "aggressive_r1")
	}
	if got := round.Contributions[agent.AgentRoleConservativeRisk]; got != "conservative_r1" {
		t.Fatalf("conservative contribution = %q, want %q", got, "conservative_r1")
	}
	if got := round.Contributions[agent.AgentRoleNeutralRisk]; got != "neutral_r1" {
		t.Fatalf("neutral contribution = %q, want %q", got, "neutral_r1")
	}
	if state.RiskDebate.FinalSignal == "" {
		t.Fatal("FinalSignal is empty, want non-empty")
	}
}

// TestRiskDebateRoundsClampedToOne verifies that rounds < 1 are clamped to 1
// and the debate still executes correctly.
func TestRiskDebateRoundsClampedToOne(t *testing.T) {
	callLog := []string{}
	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		newRiskManagerStub(&callLog),
		0,
		slog.Default(),
	)

	state := &agent.PipelineState{}

	if err := rd.Execute(context.Background(), state); err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	if got := len(state.RiskDebate.Rounds); got != 1 {
		t.Fatalf("rounds count = %d, want 1", got)
	}
	if got := len(callLog); got != 4 {
		t.Fatalf("call count = %d, want 4; log = %v", got, callLog)
	}
}

// TestRiskDebateAggressiveError verifies that an error from the aggressive
// node stops execution and propagates the error.
func TestRiskDebateAggressiveError(t *testing.T) {
	callLog := []string{}
	aggErr := errors.New("aggressive failure")
	agg := &stubNode{
		name:    "aggressive_risk",
		role:    agent.AgentRoleAggressiveRisk,
		phase:   agent.PhaseRiskDebate,
		callLog: &callLog,
		execFn:  func(_ context.Context, _ *agent.PipelineState) error { return aggErr },
	}

	rd := NewRiskDebate(
		agg,
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		newRiskManagerStub(&callLog),
		2,
		slog.Default(),
	)

	state := &agent.PipelineState{}

	err := rd.Execute(context.Background(), state)
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !errors.Is(err, aggErr) {
		t.Fatalf("error = %v, want wrapped %v", err, aggErr)
	}

	// Only aggressive should have been called.
	if len(callLog) != 1 {
		t.Fatalf("call count = %d, want 1; log = %v", len(callLog), callLog)
	}
}

// TestRiskDebateConservativeError verifies that an error from the conservative
// node stops execution after aggressive runs.
func TestRiskDebateConservativeError(t *testing.T) {
	callLog := []string{}
	consErr := errors.New("conservative failure")
	cons := &stubNode{
		name:    "conservative_risk",
		role:    agent.AgentRoleConservativeRisk,
		phase:   agent.PhaseRiskDebate,
		callLog: &callLog,
		execFn:  func(_ context.Context, _ *agent.PipelineState) error { return consErr },
	}

	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		cons,
		newNeutralStub(&callLog),
		newRiskManagerStub(&callLog),
		2,
		slog.Default(),
	)

	state := &agent.PipelineState{}

	err := rd.Execute(context.Background(), state)
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !errors.Is(err, consErr) {
		t.Fatalf("error = %v, want wrapped %v", err, consErr)
	}

	// Aggressive and conservative should have been called.
	if len(callLog) != 2 {
		t.Fatalf("call count = %d, want 2; log = %v", len(callLog), callLog)
	}
}

// TestRiskDebateNeutralError verifies that an error from the neutral node
// stops execution after aggressive and conservative run.
func TestRiskDebateNeutralError(t *testing.T) {
	callLog := []string{}
	neutErr := errors.New("neutral failure")
	neut := &stubNode{
		name:    "neutral_risk",
		role:    agent.AgentRoleNeutralRisk,
		phase:   agent.PhaseRiskDebate,
		callLog: &callLog,
		execFn:  func(_ context.Context, _ *agent.PipelineState) error { return neutErr },
	}

	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		newConservativeStub(&callLog),
		neut,
		newRiskManagerStub(&callLog),
		2,
		slog.Default(),
	)

	state := &agent.PipelineState{}

	err := rd.Execute(context.Background(), state)
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !errors.Is(err, neutErr) {
		t.Fatalf("error = %v, want wrapped %v", err, neutErr)
	}

	// Aggressive, conservative, and neutral should have been called.
	if len(callLog) != 3 {
		t.Fatalf("call count = %d, want 3; log = %v", len(callLog), callLog)
	}
}

// TestRiskDebateManagerError verifies that an error from the manager node is
// propagated after all rounds complete.
func TestRiskDebateManagerError(t *testing.T) {
	callLog := []string{}
	mgrErr := errors.New("manager failure")
	mgr := &stubNode{
		name:    "risk_manager",
		role:    agent.AgentRoleRiskManager,
		phase:   agent.PhaseRiskDebate,
		callLog: &callLog,
		execFn:  func(_ context.Context, _ *agent.PipelineState) error { return mgrErr },
	}

	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		mgr,
		1,
		slog.Default(),
	)

	state := &agent.PipelineState{}

	err := rd.Execute(context.Background(), state)
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !errors.Is(err, mgrErr) {
		t.Fatalf("error = %v, want wrapped %v", err, mgrErr)
	}

	// All 3 debaters + manager should have been called.
	wantCalls := []string{"aggressive_risk", "conservative_risk", "neutral_risk", "risk_manager"}
	if len(callLog) != len(wantCalls) {
		t.Fatalf("call count = %d, want %d; log = %v", len(callLog), len(wantCalls), callLog)
	}
}

// TestRiskDebateContextCanceled verifies that a canceled context stops
// execution before starting the next round.
func TestRiskDebateContextCanceled(t *testing.T) {
	callLog := []string{}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after first round completes.
	agg := &stubNode{
		name:    "aggressive_risk",
		role:    agent.AgentRoleAggressiveRisk,
		phase:   agent.PhaseRiskDebate,
		callLog: &callLog,
		execFn: func(_ context.Context, state *agent.PipelineState) error {
			rounds := state.RiskDebate.Rounds
			if len(rounds) > 0 {
				cur := &state.RiskDebate.Rounds[len(rounds)-1]
				if cur.Contributions == nil {
					cur.Contributions = make(map[agent.AgentRole]string)
				}
				cur.Contributions[agent.AgentRoleAggressiveRisk] = fmt.Sprintf("aggressive_r%d", cur.Number)
				// Cancel after completing the first round's aggressive call.
				if cur.Number == 1 {
					cancel()
				}
			}
			return nil
		},
	}

	rd := NewRiskDebate(
		agg,
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		newRiskManagerStub(&callLog),
		3,
		slog.Default(),
	)

	state := &agent.PipelineState{}

	err := rd.Execute(ctx, state)
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}

	// Round 1 should complete (aggressive + conservative + neutral), then
	// context check before round 2 should fail.
	if got := len(state.RiskDebate.Rounds); got != 1 {
		t.Fatalf("rounds = %d, want 1 (context should stop before round 2)", got)
	}
}

// TestNewRiskDebateNilLogger verifies that a nil logger defaults to
// slog.Default() without panicking.
func TestNewRiskDebateNilLogger(t *testing.T) {
	callLog := []string{}
	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		newRiskManagerStub(&callLog),
		1,
		nil,
	)
	if rd == nil {
		t.Fatal("NewRiskDebate() returned nil")
	}
}

// TestRiskDebateNilNodes verifies that Execute returns a descriptive error
// when any required node is nil, matching Pipeline's fail-fast pattern.
func TestRiskDebateNilNodes(t *testing.T) {
	callLog := []string{}
	agg := newAggressiveStub(&callLog)
	cons := newConservativeStub(&callLog)
	neut := newNeutralStub(&callLog)
	mgr := newRiskManagerStub(&callLog)

	tests := []struct {
		name         string
		aggressive   agent.Node
		conservative agent.Node
		neutral      agent.Node
		manager      agent.Node
		wantErr      string
	}{
		{
			name:         "nil aggressive",
			aggressive:   nil,
			conservative: cons,
			neutral:      neut,
			manager:      mgr,
			wantErr:      "debate/risk_debate: nil aggressive node",
		},
		{
			name:         "nil conservative",
			aggressive:   agg,
			conservative: nil,
			neutral:      neut,
			manager:      mgr,
			wantErr:      "debate/risk_debate: nil conservative node",
		},
		{
			name:         "nil neutral",
			aggressive:   agg,
			conservative: cons,
			neutral:      nil,
			manager:      mgr,
			wantErr:      "debate/risk_debate: nil neutral node",
		},
		{
			name:         "nil manager",
			aggressive:   agg,
			conservative: cons,
			neutral:      neut,
			manager:      nil,
			wantErr:      "debate/risk_debate: nil manager node",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rd := NewRiskDebate(tt.aggressive, tt.conservative, tt.neutral, tt.manager, 1, slog.Default())
			err := rd.Execute(context.Background(), &agent.PipelineState{})
			if err == nil {
				t.Fatal("Execute() error = nil, want non-nil")
			}
			if got := err.Error(); got != tt.wantErr {
				t.Fatalf("error = %q, want %q", got, tt.wantErr)
			}
		})
	}
}

// TestRiskDebateNilManagerRole verifies that Role() returns an empty string
// when manager is nil instead of panicking.
func TestRiskDebateNilManagerRole(t *testing.T) {
	callLog := []string{}
	rd := NewRiskDebate(
		newAggressiveStub(&callLog),
		newConservativeStub(&callLog),
		newNeutralStub(&callLog),
		nil,
		1,
		slog.Default(),
	)
	if got := rd.Role(); got != "" {
		t.Fatalf("Role() = %q, want empty string for nil manager", got)
	}
}

// Verify RiskDebate satisfies the agent.Node interface at compile time.
var _ agent.Node = (*RiskDebate)(nil)
