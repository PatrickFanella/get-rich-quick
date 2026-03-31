package papervalidation

import (
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/backtest"
)

// ValidationReport is the daily automated report produced during the 60-day
// paper trading validation period. It summarises current performance, metric
// pass/fail status, and the go/no-go decision.
type ValidationReport struct {
	ReportDate       time.Time       `json:"report_date"`
	PaperStartDate   time.Time       `json:"paper_start_date"`
	ElapsedDays      int             `json:"elapsed_days"`
	DaysRemaining    int             `json:"days_remaining"`
	Metrics          []MetricResult  `json:"metrics"`
	AllMetricsPassed bool            `json:"all_metrics_passed"`
	GoDecision       bool            `json:"go_decision"`
	Decision         string          `json:"decision"`
	TransitionPlan   *TransitionPlan `json:"transition_plan,omitempty"`
}

// TransitionPlan describes the phased rollout when a GO decision is reached.
type TransitionPlan struct {
	InitialPositionPct float64 `json:"initial_position_pct"`
	Description        string  `json:"description"`
}

// GenerateReport produces a ValidationReport for the given date using the
// supplied metrics and analytics.
func GenerateReport(
	metrics backtest.Metrics,
	analytics backtest.TradeAnalytics,
	thresholds Thresholds,
	paperStart, reportDate time.Time,
) ValidationReport {
	result := Validate(metrics, analytics, thresholds, paperStart, reportDate)

	remaining := thresholds.MinCalendarDays - result.ElapsedDays
	if remaining < 0 {
		remaining = 0
	}

	decision := "NO-GO"
	var plan *TransitionPlan
	if result.GoDecision {
		decision = "GO"
		plan = &TransitionPlan{
			InitialPositionPct: 25.0,
			Description:        "Run live at 25% of paper position sizes alongside ongoing paper trading for 2 weeks with explicit revert criteria; if criteria are satisfied, scale weekly to 50%, 75%, then 100%.",
		}
	}

	return ValidationReport{
		ReportDate:       reportDate,
		PaperStartDate:   paperStart,
		ElapsedDays:      result.ElapsedDays,
		DaysRemaining:    remaining,
		Metrics:          result.Metrics,
		AllMetricsPassed: result.AllPassed,
		GoDecision:       result.GoDecision,
		Decision:         decision,
		TransitionPlan:   plan,
	}
}
