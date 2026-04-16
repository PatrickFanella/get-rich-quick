package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

const sentimentBatchSize = 10

// SentimentResult aggregates LLM-derived sentiment for a ticker from Reddit posts.
type SentimentResult struct {
	Mentions int
	Bullish  int
	Bearish  int
	Neutral  int
}

// postSentiment is the per-post LLM response.
type postSentiment struct {
	MentionsTicker bool   `json:"mentions_ticker"` // does post reference the target ticker?
	Sentiment      string `json:"sentiment"`       // bullish, bearish, neutral
}

func sentimentSystemPrompt(ticker string) string {
	return fmt.Sprintf(`You are a financial social-media sentiment classifier.
For each Reddit post, determine:
1. Whether it mentions or is clearly about the stock ticker %s (use the symbol, not just the company name).
2. If it does mention the ticker, classify the overall sentiment toward %s as "bullish", "bearish", or "neutral".

Respond with a JSON array. Each element must have:
- "mentions_ticker": true/false
- "sentiment": "bullish" | "bearish" | "neutral" (only meaningful when mentions_ticker is true)

Return ONLY the JSON array.`, ticker, ticker)
}

// ScorePosts runs LLM triage on posts to extract sentiment about a specific ticker.
// Posts are batched for efficiency. Returns aggregated counts.
func ScorePosts(ctx context.Context, provider llm.Provider, model, ticker string, posts []RedditPost, logger *slog.Logger) SentimentResult {
	if provider == nil || len(posts) == 0 {
		return SentimentResult{}
	}
	if logger == nil {
		logger = slog.Default()
	}

	var result SentimentResult
	for i := 0; i < len(posts); i += sentimentBatchSize {
		end := i + sentimentBatchSize
		if end > len(posts) {
			end = len(posts)
		}
		batch := posts[i:end]
		r := scoreBatch(ctx, provider, model, ticker, batch, logger)
		result.Mentions += r.Mentions
		result.Bullish += r.Bullish
		result.Bearish += r.Bearish
		result.Neutral += r.Neutral
	}
	return result
}

func scoreBatch(ctx context.Context, provider llm.Provider, model, ticker string, batch []RedditPost, logger *slog.Logger) SentimentResult {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Classify each post for ticker %s. Return a JSON array with one object per post, in order.\n\n", ticker)
	for i, p := range batch {
		title := p.Title
		body := p.Body
		if len(body) > 300 {
			body = body[:300] + "..."
		}
		fmt.Fprintf(&sb, "%d. [r/%s] %s\n", i+1, p.Subreddit, title)
		if body != "" {
			fmt.Fprintf(&sb, "   %s\n", body)
		}
	}

	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: "system", Content: sentimentSystemPrompt(ticker)},
			{Role: "user", Content: sb.String()},
		},
		ResponseFormat: &llm.ResponseFormat{Type: llm.ResponseFormatJSONObject},
	})
	if err != nil {
		logger.Warn("reddit/sentiment: LLM call failed", slog.Any("error", err))
		return SentimentResult{}
	}

	content := strings.TrimSpace(resp.Content)
	// Strip markdown fences if present.
	if strings.HasPrefix(content, "```") {
		if idx := strings.Index(content[3:], "\n"); idx >= 0 {
			content = content[3+idx+1:]
		}
		if idx := strings.LastIndex(content, "```"); idx >= 0 {
			content = content[:idx]
		}
		content = strings.TrimSpace(content)
	}

	var sentiments []postSentiment
	if err := json.Unmarshal([]byte(content), &sentiments); err != nil {
		// Try wrapper: {"results": [...]}
		var wrapper struct {
			Results []postSentiment `json:"results"`
		}
		if err2 := json.Unmarshal([]byte(content), &wrapper); err2 != nil {
			logger.Warn("reddit/sentiment: failed to parse LLM response",
				slog.Any("error", err),
				slog.String("content", content[:min(200, len(content))]),
			)
			return SentimentResult{}
		}
		sentiments = wrapper.Results
	}

	var result SentimentResult
	for _, s := range sentiments {
		if !s.MentionsTicker {
			continue
		}
		result.Mentions++
		switch strings.ToLower(s.Sentiment) {
		case "bullish":
			result.Bullish++
		case "bearish":
			result.Bearish++
		default:
			result.Neutral++
		}
	}
	return result
}
