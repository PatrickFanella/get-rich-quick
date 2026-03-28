# 60-Day Paper Trading Validation Plan

## Purpose

This document defines the validation criteria, automated reporting process, and
go/no-go decision framework for the mandatory 60-day paper trading period. No
live capital may be deployed until every criterion listed below is satisfied.

## Validation Period

- **Duration:** 60 calendar days minimum.
- **Start date:** first paper trade execution.
- **End date:** 60 calendar days after start, provided all criteria are met.

## Performance Thresholds

| Metric              | Threshold      | Description                                     |
| ------------------- | -------------- | ----------------------------------------------- |
| Sharpe Ratio        | > 1.0          | Annualised risk-adjusted return (risk-free = 0)  |
| Max Drawdown        | < 15%          | Worst peak-to-trough drawdown                    |
| Win Rate            | > 40%          | Fraction of round-trip trades with positive PnL  |
| Profit Factor       | > 1.5          | Gross profits / gross losses                     |
| Round-Trip Trades   | ≥ 20           | Minimum closed trades for statistical validity   |

All five thresholds must be satisfied simultaneously at the end of the 60-day
window for a **GO** decision.

## Automated Daily Validation Report

A `ValidationReport` is generated daily and contains:

1. **Report metadata** — report date, paper trading start date, elapsed days,
   days remaining.
2. **Metric results** — each metric's current value, required threshold, and
   pass/fail status.
3. **Overall status** — aggregate pass/fail.
4. **Go/No-Go decision** — whether the strategy is approved for live trading.

Reports are produced by the `papervalidation` package
(`internal/papervalidation`).

## Go / No-Go Checklist

A strategy receives a **GO** decision when **all** of the following are true:

- [ ] At least 60 calendar days have elapsed since the first paper trade.
- [ ] Sharpe ratio > 1.0.
- [ ] Max drawdown < 15%.
- [ ] Win rate > 40%.
- [ ] Profit factor > 1.5.
- [ ] At least 20 round-trip (closed) trades executed.

If any criterion is not met, the decision is **NO-GO** and paper trading
continues.

## Transition Plan

When a GO decision is reached:

1. **Start at 25%** of paper position sizes for live trading.
2. Run live alongside paper for an additional 2 weeks.
3. If live performance tracks paper within acceptable tolerance, scale to 50 %,
   then 75 %, then 100 % in weekly increments.
4. If live performance deviates materially (> 2× paper max drawdown or Sharpe
   drops below 0.5), revert to paper-only and re-evaluate.

## Definition of Done

- [x] Validation plan documented (this document).
- [x] Automated reporting implemented (`internal/papervalidation`).
- [x] Criteria defined and agreed upon.
