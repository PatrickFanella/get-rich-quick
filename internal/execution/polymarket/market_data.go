package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
)

// GetMarketData fetches complete market metadata and order-book state for the
// given Polymarket market slug and maps it to an agent.PredictionMarketData.
//
// API calls made:
//  1. GET /markets?market_slug={slug}   — market metadata + token IDs
//  2. GET /prices-history?market=…      — current YES/NO mid-prices
//  3. GET /book?token_id={yesTokenID}  — best bid/ask for YES token
//  4. GET /book?token_id={noTokenID}   — best bid/ask for NO token
func (c *Client) GetMarketData(ctx context.Context, slug string) (*agent.PredictionMarketData, error) {
	market, err := c.fetchMarketBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	yesPrice, noPrice, err := c.fetchMidPrices(ctx, market.ConditionID, market.YesTokenID, market.NoTokenID)
	if err != nil {
		// Non-fatal: return what we have with zero prices.
		c.getLogger().Warn("polymarket: failed to fetch mid-prices", "slug", slug, "error", err)
	}

	yesBook, _ := c.fetchBook(ctx, market.YesTokenID)
	noBook, _ := c.fetchBook(ctx, market.NoTokenID)

	var endDate *time.Time
	if !market.EndDateISO.IsZero() {
		t := market.EndDateISO
		endDate = &t
	}

	pmd := &agent.PredictionMarketData{
		Slug:               market.MarketSlug,
		Question:           market.Question,
		Description:        market.Description,
		ResolutionCriteria: market.ResolutionCriteria,
		EndDate:            endDate,
		ResolutionSource:   market.ResolutionSource,
		YesPrice:           yesPrice,
		NoPrice:            noPrice,
		Volume24h:          market.Volume24h,
		Liquidity:          market.Liquidity,
		OpenInterest:       market.OpenInterest,
		ConditionID:        market.ConditionID,
		YesTokenID:         market.YesTokenID,
		NoTokenID:          market.NoTokenID,
		BestBidYes:         yesBook.bestBid,
		BestAskYes:         yesBook.bestAsk,
		BestBidNo:          noBook.bestBid,
		BestAskNo:          noBook.bestAsk,
	}
	if pmd.BestAskYes > 0 {
		pmd.SpreadYes = pmd.BestAskYes - pmd.BestBidYes
	}

	return pmd, nil
}

// ---------------------------------------------------------------------------
// internal API response types
// ---------------------------------------------------------------------------

type clobMarket struct {
	MarketSlug         string    `json:"market_slug"`
	Question           string    `json:"question"`
	Description        string    `json:"description"`
	ResolutionCriteria string    `json:"resolution_criteria"`
	ResolutionSource   string    `json:"resolution_source"`
	EndDateISO         time.Time `json:"end_date_iso"`
	ConditionID        string    `json:"condition_id"`
	Tokens             []struct {
		Outcome string `json:"outcome"` // "Yes" or "No"
		TokenID string `json:"token_id"`
	} `json:"tokens"`
	Volume24h    float64 `json:"volume_24hr"`
	Liquidity    float64 `json:"liquidity"`
	OpenInterest float64 `json:"open_interest"`

	// resolved during fetch
	YesTokenID string
	NoTokenID  string
}

type bookSide struct {
	bestBid float64
	bestAsk float64
}

type clobBookResponse struct {
	Bids []struct {
		Price float64 `json:"price,string"`
		Size  float64 `json:"size,string"`
	} `json:"bids"`
	Asks []struct {
		Price float64 `json:"price,string"`
		Size  float64 `json:"size,string"`
	} `json:"asks"`
}

// ---------------------------------------------------------------------------
// fetch helpers
// ---------------------------------------------------------------------------

func (c *Client) fetchMarketBySlug(ctx context.Context, slug string) (*clobMarket, error) {
	params := url.Values{"market_slug": []string{slug}}
	body, err := c.Get(ctx, "/markets", params)
	if err != nil {
		return nil, fmt.Errorf("polymarket: fetch market %q: %w", slug, err)
	}

	// The CLOB API returns {"data": [...]} for list endpoints.
	var resp struct {
		Data []clobMarket `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("polymarket: parse market list: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("polymarket: market not found: %q", slug)
	}

	m := &resp.Data[0]
	for _, tok := range m.Tokens {
		switch tok.Outcome {
		case "Yes":
			m.YesTokenID = tok.TokenID
		case "No":
			m.NoTokenID = tok.TokenID
		}
	}
	if m.MarketSlug == "" {
		m.MarketSlug = slug
	}
	return m, nil
}

func (c *Client) fetchMidPrices(ctx context.Context, _, yesTokenID, noTokenID string) (yesPrice, noPrice float64, err error) {
	if yesTokenID == "" || noTokenID == "" {
		return 0, 0, nil
	}

	params := url.Values{"token_ids": []string{yesTokenID + "," + noTokenID}}
	body, err := c.Get(ctx, "/prices", params)
	if err != nil {
		return 0, 0, err
	}

	// Response: {"<tokenID>": "0.72", ...}
	var prices map[string]json.Number
	if err := json.Unmarshal(body, &prices); err != nil {
		return 0, 0, fmt.Errorf("polymarket: parse prices: %w", err)
	}

	if v, ok := prices[yesTokenID]; ok {
		if f, err := v.Float64(); err == nil {
			yesPrice = f
		}
	}
	if v, ok := prices[noTokenID]; ok {
		if f, err := v.Float64(); err == nil {
			noPrice = f
		}
	}
	return yesPrice, noPrice, nil
}

func (c *Client) fetchBook(ctx context.Context, tokenID string) (bookSide, error) {
	if tokenID == "" {
		return bookSide{}, nil
	}
	params := url.Values{"token_id": []string{tokenID}}
	body, err := c.Get(ctx, "/book", params)
	if err != nil {
		return bookSide{}, err
	}

	var resp clobBookResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return bookSide{}, fmt.Errorf("polymarket: parse book: %w", err)
	}

	var best bookSide
	if len(resp.Bids) > 0 {
		best.bestBid = resp.Bids[0].Price
	}
	if len(resp.Asks) > 0 {
		best.bestAsk = resp.Asks[0].Price
	}
	return best, nil
}
