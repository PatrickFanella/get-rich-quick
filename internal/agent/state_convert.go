package agent

// PipelineStateFromView reconstructs a mutable *PipelineState from a StateView
// snapshot. The returned state has an initialised mutex but empty decisions map
// and nil Errors slice.
func PipelineStateFromView(view StateView) *PipelineState {
	state := &PipelineState{
		PipelineRunID:  view.PipelineRunID,
		StrategyID:     view.StrategyID,
		Ticker:         view.Ticker,
		AnalystReports: make(map[AgentRole]string, len(view.AnalystReports)),
		ResearchDebate: view.ResearchDebate,
		TradingPlan:    view.TradingPlan,
		RiskDebate:     view.RiskDebate,
		FinalSignal:    view.FinalSignal,
		LLMCacheStats:  view.LLMCacheStats,
	}
	for role, report := range view.AnalystReports {
		state.AnalystReports[role] = report
	}
	return state
}
