package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

const (
	sentimentBatchSize    = 10
	sentimentMaxTokens    = 4096
	sentimentConcurrency  = 3
	sentimentMaxRetries   = 1
)

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
// Posts are batched for efficiency and processed with bounded concurrency.
// Returns aggregated counts.
func ScorePosts(ctx context.Context, provider llm.Provider, model, ticker string, posts []RedditPost, logger *slog.Logger) SentimentResult {
	if provider == nil || len(posts) == 0 {
		return SentimentResult{}
	}
	if logger == nil {
		logger = slog.Default()
	}

	// Build batches up front.
	type indexedBatch struct {
		index int
		batch []RedditPost
	}
	var batches []indexedBatch
	for i := 0; i < len(posts); i += sentimentBatchSize {
		end := i + sentimentBatchSize
		if end > len(posts) {
			end = len(posts)
		}
		batches = append(batches, indexedBatch{index: i / sentimentBatchSize, batch: posts[i:end]})
	}

	totalBatches := len(batches)

	// Process batches with bounded concurrency.
	sem := make(chan struct{}, sentimentConcurrency)
	var (
		mu     sync.Mutex
		result SentimentResult
	)

	var wg sync.WaitGroup
	for _, b := range batches {
		wg.Add(1)
		go func(ib indexedBatch) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r := scoreBatch(ctx, provider, model, ticker, ib.batch, ib.index, totalBatches, logger)
			mu.Lock()
			result.Mentions += r.Mentions
			result.Bullish += r.Bullish
			result.Bearish += r.Bearish
			result.Neutral += r.Neutral
			mu.Unlock()
		}(b)
	}
	wg.Wait()

	return result
}

func scoreBatch(ctx context.Context, provider llm.Provider, model, ticker string, batch []RedditPost, batchIdx, totalBatches int, logger *slog.Logger) SentimentResult {
	prompt := buildBatchPrompt(ticker, batch)

	for attempt := 0; attempt <= sentimentMaxRetries; attempt++ {
		sysPrompt := sentimentSystemPrompt(ticker)
		if attempt > 0 {
			// On retry, explicitly instruct the model not to use thinking mode
			// which can cause empty content with Qwen3 models.
			sysPrompt += "\n\nIMPORTANT: Do NOT use <think> tags. Respond directly with the JSON array."
		}

		resp, err := provider.Complete(ctx, llm.CompletionRequest{
			Model: model,
			Messages: []llm.Message{
				{Role: "system", Content: sysPrompt},
				{Role: "user", Content: prompt},
			},
			MaxTokens:      sentimentMaxTokens,
			ResponseFormat: &llm.ResponseFormat{Type: llm.ResponseFormatJSONObject},
		})
		if err != nil {
			logger.Warn("reddit/sentiment: LLM call failed",
				slog.Int("batch", batchIdx+1),
				slog.Int("total_batches", totalBatches),
				slog.Int("attempt", attempt+1),
				slog.Any("error", err),
			)
			return SentimentResult{}
		}

		content := cleanContent(resp.Content)
		if content == "" {
			logger.Warn("reddit/sentiment: empty LLM response, retrying",
				slog.Int("batch", batchIdx+1),
				slog.Int("total_batches", totalBatches),
				slog.Int("attempt", attempt+1),
			)
			continue
		}

		result, ok := parseSentimentResponse(content)
		if !ok {
			logger.Warn("reddit/sentiment: failed to parse LLM response",
				slog.Int("batch", batchIdx+1),
				slog.Int("total_batches", totalBatches),
				slog.Int("attempt", attempt+1),
				slog.String("content", content[:min(200, len(content))]),
			)
			return SentimentResult{}
		}
		return result
	}

	logger.Warn("reddit/sentiment: exhausted retries with empty responses",
		slog.Int("batch", batchIdx+1),
		slog.Int("total_batches", totalBatches),
	)
	return SentimentResult{}
}

func buildBatchPrompt(ticker string, batch []RedditPost) string {
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
	return sb.String()
}

func cleanContent(raw string) string {
	content := strings.TrimSpace(raw)
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
	return content
}

func parseSentimentResponse(content string) (SentimentResult, bool) {
	var sentiments []postSentiment
	if err := json.Unmarshal([]byte(content), &sentiments); err != nil {
		// Try wrapper: {"results": [...]}
		var wrapper struct {
			Results []postSentiment `json:"results"`
		}
		if err2 := json.Unmarshal([]byte(content), &wrapper); err2 != nil {
			return SentimentResult{}, false
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
	return result, true
}
