# ADR-008: Correlated asset exposure management

- **Status:** accepted (deferred implementation)
- **Date:** 2026-03-27
- **Deciders:** Engineering
- **Technical Story:** [#111](https://github.com/PatrickFanella/get-rich-quick/issues/111)

## Context

Current position limits (`internal/risk/engine_impl.go`) check per-position exposure (20%), per-market exposure (50%, 5% for Polymarket), total exposure (100%), and concurrent position count (10). However, there is no quantitative check for correlation between positions. Two positions in highly correlated assets (e.g., AAPL and MSFT) could produce concentrated exposure that the per-position limits do not catch.

The risk debate agents (conservative and neutral analysts in `internal/agent/risk/`) are prompted to flag correlation risk qualitatively, but this is LLM judgment, not quantitative enforcement.

## Decision

**Defer quantitative correlation enforcement.** Rely on LLM-based qualitative judgment for now.

### Rationale

1. **Scale doesn't justify the cost.** Paper trading will run 1-5 strategies with at most 10 concurrent positions. Correlation-driven blowups require larger, more concentrated portfolios.
2. **Correlation matrices require infrastructure.** Computing rolling correlations needs historical return data, regular recomputation (daily), and a data pipeline that doesn't yet exist for this purpose.
3. **Existing limits cap concentration.** 20% per position and 10 max concurrent positions mean the worst case is two 20% positions in correlated names — a 40% effective exposure that the circuit breaker (10% max drawdown) would catch before catastrophic loss.
4. **LLM risk debate already covers this.** The conservative and neutral risk analysts explicitly evaluate correlation risk in their prompts and can reject correlated position proposals.

### Lightweight alternative (if needed later)

If correlation concentration causes a realized loss, implement sector-based grouping as an intermediate step before full correlation matrices:

- Classify tickers by GICS sector (using data provider metadata)
- Cap per-sector exposure at 40% (configurable in `RiskLimits`)
- No correlation matrix needed — sector classification is a simple lookup

### Revisit trigger

Revisit this decision when:

- Running >20 concurrent strategies
- A realized loss is attributable to correlated position concentration
- Sector concentration exceeds 60% of portfolio in backtesting

## Consequences

- No additional infrastructure or data pipeline needed for Phase 5.
- Correlation risk is managed qualitatively via LLM risk debate, which may miss subtle correlations (e.g., commodities exposure through equity positions).
- The sector-based fallback provides a clear upgrade path without requiring a full correlation engine.
