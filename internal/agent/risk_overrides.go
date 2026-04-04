package agent

import (
	"fmt"
	"math"
	"strings"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// ApplyStrategyRiskOverrides applies post-pipeline risk adjustments to the
// trading plan and signal based on the strategy's risk configuration.
// It can scale stop-loss/take-profit targets and gate trades below a minimum
// confidence threshold (flipping the signal to hold).
func ApplyStrategyRiskOverrides(state *PipelineState, signal domain.PipelineSignal, strategyConfig *StrategyConfig) domain.PipelineSignal {
	if state == nil || strategyConfig == nil || strategyConfig.RiskConfig == nil {
		return signal
	}

	riskConfig := strategyConfig.RiskConfig
	plan := state.TradingPlan
	if riskConfig.StopLossMultiplier != nil || riskConfig.TakeProfitMultiplier != nil {
		AdjustTradingPlanTargets(&plan, signal, riskConfig)
		state.TradingPlan = plan
	}

	if riskConfig.MinConfidence != nil && state.FinalSignal.Confidence < *riskConfig.MinConfidence {
		signal = domain.PipelineSignalHold
		state.FinalSignal.Signal = signal
		state.TradingPlan.Action = signal
		state.TradingPlan.PositionSize = 0
		state.TradingPlan.Rationale = appendRationale(state.TradingPlan.Rationale, fmt.Sprintf(
			"Signal confidence %.2f fell below configured minimum %.2f.",
			state.FinalSignal.Confidence,
			*riskConfig.MinConfidence,
		))
	}

	return signal
}

// AdjustTradingPlanTargets scales the stop-loss and take-profit distances on the
// trading plan using the configured multipliers. A multiplier of 1.0 leaves the
// target unchanged; >1 widens it, <1 tightens it.
func AdjustTradingPlanTargets(plan *TradingPlan, signal domain.PipelineSignal, cfg *StrategyRiskConfig) {
	if plan == nil || plan.EntryPrice <= 0 {
		return
	}

	if cfg.StopLossMultiplier != nil && plan.StopLoss > 0 {
		distance := math.Abs(plan.EntryPrice - plan.StopLoss)
		scaled := distance * *cfg.StopLossMultiplier
		switch signal {
		case domain.PipelineSignalBuy:
			plan.StopLoss = math.Max(0, plan.EntryPrice-scaled)
		case domain.PipelineSignalSell:
			plan.StopLoss = plan.EntryPrice + scaled
		}
	}

	if cfg.TakeProfitMultiplier != nil && plan.TakeProfit > 0 {
		distance := math.Abs(plan.TakeProfit - plan.EntryPrice)
		scaled := distance * *cfg.TakeProfitMultiplier
		switch signal {
		case domain.PipelineSignalBuy:
			plan.TakeProfit = plan.EntryPrice + scaled
		case domain.PipelineSignalSell:
			plan.TakeProfit = math.Max(0, plan.EntryPrice-scaled)
		}
	}
}

// ApplyStrategyRiskOverridesToResult applies post-pipeline risk adjustments
// directly to a RunResult, avoiding the lossy StateView → PipelineState
// round-trip. Mutations to TradingPlan, FinalSignal, and Signal are all
// visible on the result after this call.
func ApplyStrategyRiskOverridesToResult(result *RunResult, strategyConfig *StrategyConfig) {
	if result == nil || strategyConfig == nil || strategyConfig.RiskConfig == nil {
		return
	}

	riskConfig := strategyConfig.RiskConfig
	if riskConfig.StopLossMultiplier != nil || riskConfig.TakeProfitMultiplier != nil {
		AdjustTradingPlanTargets(&result.State.TradingPlan, result.Signal, riskConfig)
	}

	if riskConfig.MinConfidence != nil && result.State.FinalSignal.Confidence < *riskConfig.MinConfidence {
		result.Signal = domain.PipelineSignalHold
		result.State.FinalSignal.Signal = result.Signal
		result.State.TradingPlan.Action = result.Signal
		result.State.TradingPlan.PositionSize = 0
		result.State.TradingPlan.Rationale = appendRationale(result.State.TradingPlan.Rationale, fmt.Sprintf(
			"Signal confidence %.2f fell below configured minimum %.2f.",
			result.State.FinalSignal.Confidence,
			*riskConfig.MinConfidence,
		))
	}
}

func appendRationale(existing, addition string) string {
	existing = strings.TrimSpace(existing)
	addition = strings.TrimSpace(addition)
	if existing == "" {
		return addition
	}
	if addition == "" {
		return existing
	}
	return existing + " " + addition
}
