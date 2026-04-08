package analysts

import (
	"fmt"
	"strings"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
)

// PolymarketFundamentalsSystemPrompt instructs the LLM to perform prediction
// market analysis in place of the standard equity fundamentals analysis.
const PolymarketFundamentalsSystemPrompt = `You are a prediction market analyst. Your job is to assess whether a prediction market is correctly priced given its resolution criteria, current YES/NO prices, liquidity, and time to resolution.

## Analysis Framework

### Resolution Assessment
- Evaluate how clearly the resolution criteria are defined. Ambiguous criteria carry tail risk.
- Identify edge cases or scenarios where resolution might be contested.
- Assess the credibility and objectivity of the resolution source.

### Price Assessment
- Compare current YES price (= market-implied probability) to your probability estimate.
- A YES price of 0.60 means the market assigns 60% probability to the event resolving YES.
- Identify mispricing: is the market over- or under-estimating the probability?

### Information & Edge
- Consider what information may not be priced in yet.
- Assess whether there is an information asymmetry advantage or disadvantage.

### Time Decay & Liquidity Risk
- Markets near resolution carry higher gamma: small new information causes large price swings.
- Low liquidity means wide effective spread and difficulty exiting quickly.
- Thin order books amplify slippage risk.

## Output Format

Produce a structured report:
1. **Resolution Analysis** — criteria clarity, edge cases, resolution source credibility
2. **Price Assessment** — current YES/NO price interpretation, your probability estimate, mispricing direction
3. **Information Edge** — known vs. unknown information, asymmetry assessment
4. **Time-to-Resolution Risk** — remaining duration, gamma risk, exit risk
5. **Liquidity Assessment** — volume, open interest, order book depth, spread
6. **Overall Assessment** — conviction level (low/medium/high), directional bias (YES/NO/neutral), key risks

Be precise with numbers. Reference actual values from the data provided.`

// PolymarketMarketAnalystNote is appended to the MarketAnalyst user prompt for
// prediction market strategies to frame technical indicators correctly.
const PolymarketMarketAnalystNote = `
## Prediction Market Context

This is a prediction market. Price (0–1) represents market-implied probability, not an asset price:
- Indicators such as RSI and SMA measure probability momentum, not price momentum.
- A rising RSI indicates the market is assigning increasing probability to the YES outcome.
- Standard overbought/oversold levels still apply as sentiment extremes.
- Volume spikes often precede resolution-relevant news events.
`

// PolymarketNewsAnalystNote is appended to the NewsAnalyst user prompt for
// prediction market strategies.
const PolymarketNewsAnalystNote = `
## Prediction Market Context

This is a prediction market. Analyze how each news item affects the probability of the event resolving YES or NO:
- Does this news make the YES outcome more or less likely?
- Estimate the magnitude of probability impact (small/moderate/large shift).
- Flag any news that directly concerns the resolution criteria or resolution source.
`

// FormatPolymarketFundamentalsUserPrompt builds the user prompt for the
// FundamentalsAnalyst when the pipeline is running in Polymarket mode.
func FormatPolymarketFundamentalsUserPrompt(input agent.AnalysisInput) string {
	pm := input.PredictionMarket
	if pm == nil {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Analyze the following prediction market: %s\n\n", sanitizeCell(pm.Slug))

	fmt.Fprintf(&b, "## Market Details\n\n")
	fmt.Fprintf(&b, "**Question:** %s\n\n", sanitizeCell(pm.Question))
	if pm.Description != "" {
		fmt.Fprintf(&b, "**Description:** %s\n\n", sanitizeCell(pm.Description))
	}
	if pm.ResolutionCriteria != "" {
		fmt.Fprintf(&b, "**Resolution Criteria:** %s\n\n", sanitizeCell(pm.ResolutionCriteria))
	}
	if pm.ResolutionSource != "" {
		fmt.Fprintf(&b, "**Resolution Source:** %s\n\n", sanitizeCell(pm.ResolutionSource))
	}
	if pm.EndDate != nil {
		daysLeft := int(time.Until(*pm.EndDate).Hours() / 24)
		fmt.Fprintf(&b, "**End Date:** %s (%d days remaining)\n\n", pm.EndDate.Format(time.DateOnly), daysLeft)
	}

	fmt.Fprintf(&b, "## Current Prices\n\n")
	fmt.Fprintf(&b, "| Side | Price | Implied Probability |\n")
	fmt.Fprintf(&b, "|------|-------|--------------------|\n")
	fmt.Fprintf(&b, "| YES  | %.3f | %.1f%%             |\n", pm.YesPrice, pm.YesPrice*100)
	fmt.Fprintf(&b, "| NO   | %.3f | %.1f%%             |\n", pm.NoPrice, pm.NoPrice*100)
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "## Market Depth\n\n")
	fmt.Fprintf(&b, "| Metric | Value |\n")
	fmt.Fprintf(&b, "|--------|-------|\n")
	fmt.Fprintf(&b, "| 24h Volume | $%.0f USDC |\n", pm.Volume24h)
	fmt.Fprintf(&b, "| Total Liquidity | $%.0f USDC |\n", pm.Liquidity)
	fmt.Fprintf(&b, "| Open Interest | $%.0f USDC |\n", pm.OpenInterest)
	fmt.Fprintf(&b, "\n")

	if pm.BestBidYes > 0 || pm.BestAskYes > 0 {
		fmt.Fprintf(&b, "## Order Book\n\n")
		fmt.Fprintf(&b, "| Side | Best Bid | Best Ask | Spread |\n")
		fmt.Fprintf(&b, "|------|----------|----------|--------|\n")
		fmt.Fprintf(&b, "| YES  | %.3f | %.3f | %.3f |\n", pm.BestBidYes, pm.BestAskYes, pm.SpreadYes)
		fmt.Fprintf(&b, "| NO   | %.3f | %.3f | —    |\n", pm.BestBidNo, pm.BestAskNo)
		fmt.Fprintf(&b, "\n")
	}

	b.WriteString("Provide your structured prediction market analysis report.\n")
	return b.String()
}

// polymarketSystemPromptFor returns the appropriate system prompt for
// fundamentals analysis depending on whether prediction market data is present.
func polymarketSystemPromptFor(pm bool) string {
	if pm {
		return PolymarketFundamentalsSystemPrompt
	}
	return FundamentalsAnalystSystemPrompt
}
