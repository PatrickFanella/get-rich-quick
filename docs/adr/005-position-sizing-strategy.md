# ADR-005: Position sizing strategy selection

- **Status:** accepted
- **Date:** 2026-03-27
- **Deciders:** Engineering
- **Technical Story:** [#93](https://github.com/PatrickFanella/get-rich-quick/issues/93)

## Context

Three position sizing methods are implemented in `internal/execution/position_sizing.go`:

1. **ATR-Based:** `(AccountValue * RiskPct) / (ATR * Multiplier)` — adapts to volatility, intuitive risk framing.
2. **Kelly Criterion:** `AccountValue * (WinRate - (1 - WinRate) / WinLossRatio)` — theoretically optimal but requires accurate statistics. Supports half-Kelly via `HalfKelly` flag.
3. **Fixed Fractional:** `(AccountValue * FractionPct) / PricePerShare` — simplest, most conservative.

We need to decide the default method per market type, when to switch between methods, and half-Kelly vs full-Kelly guidance.

## Decision

### Default method per market type

| Market              | Default Method        | Rationale                                                                |
| ------------------- | --------------------- | ------------------------------------------------------------------------ |
| US Equities (stock) | ATR-based             | Liquid enough for reliable ATR calculation; adapts to regime changes     |
| Crypto              | ATR-based             | High volatility makes fixed-fraction dangerous; ATR tracks regime shifts |
| Polymarket          | Fixed Fractional (2%) | Illiquid CLOB; ATR unreliable on prediction market time series           |

### Kelly Criterion usage

- **Always half-Kelly.** Full Kelly assumes perfect win-rate estimates. With LLM-generated signals, statistics will be noisy and overfit to small samples. Half-Kelly cuts maximum drawdown ~50% for ~25% less expected return.
- **Kelly is only eligible after 100+ closed trades** for the strategy. Before that threshold, insufficient statistical significance exists to trust win-rate and win/loss ratio estimates. Fall back to ATR-based or fixed fractional.
- **Kelly override requires explicit opt-in** via the strategy's `config` JSONB field. It is never the automatic default.

### Configuration

Per-strategy override is supported via the `strategy.config` JSONB field. The `PositionSizingMethod` enum and `PositionSizingParams` struct are passed through the pipeline to the order manager. No changes to the existing interface are needed.

## Consequences

- ATR-based sizing requires at least 14 bars of historical data to compute ATR. Strategies on newly listed tickers may need to fall back to fixed fractional until enough data accumulates.
- Half-Kelly is conservative but matches the system's risk philosophy (defense-in-depth). Users who want more aggressive sizing can override per-strategy, but the system will never default to full Kelly.
- The 100-trade threshold for Kelly eligibility means new strategies always start with ATR or fixed fractional, which is the safer default.
