package reddit

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	userAgent     = "get-rich-quick/1.0 social-sentiment"
	clientTimeout = 15 * time.Second
	maxBodySize   = 2 * 1024 * 1024 // 2 MiB
	fetchDelay    = 1 * time.Second // delay between subreddit fetches to respect rate limits
	maxSubreddits = 10              // safety cap
)

// StockSubreddits returns the default subreddits to scan for equity sentiment.
func StockSubreddits() []string {
	return []string{"wallstreetbets", "stocks", "investing", "options"}
}

// CryptoSubreddits returns the default subreddits to scan for crypto sentiment.
func CryptoSubreddits() []string {
	return []string{"cryptocurrency", "cryptomarkets"}
}

// RedditPost is a single post parsed from a Reddit Atom RSS feed.
type RedditPost struct {
	Title     string
	Body      string
	URL       string
	Author    string
	Subreddit string
	UpdatedAt time.Time
}

// Client fetches Reddit RSS feeds and returns parsed posts.
type Client struct {
	client *http.Client
	logger *slog.Logger
}

// NewClient creates a Reddit RSS client.
func NewClient(logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		client: &http.Client{Timeout: clientTimeout},
		logger: logger,
	}
}

// FetchSubreddits fetches posts from multiple subreddits concurrently.
// A short delay is inserted between requests to respect Reddit rate limits.
func (c *Client) FetchSubreddits(ctx context.Context, subreddits []string) []RedditPost {
	if len(subreddits) > maxSubreddits {
		subreddits = subreddits[:maxSubreddits]
	}

	var (
		mu    sync.Mutex
		posts []RedditPost
		wg    sync.WaitGroup
	)

	for i, sub := range subreddits {
		if i > 0 {
			select {
			case <-ctx.Done():
				break
			case <-time.After(fetchDelay):
			}
		}

		wg.Add(1)
		go func(sub string) {
			defer wg.Done()
			got, err := c.fetchSubreddit(ctx, sub)
			if err != nil {
				c.logger.Warn("reddit: fetch failed",
					slog.String("subreddit", sub),
					slog.Any("error", err),
				)
				return
			}
			mu.Lock()
			posts = append(posts, got...)
			mu.Unlock()
		}(sub)
	}
	wg.Wait()

	return posts
}

func (c *Client) fetchSubreddit(ctx context.Context, sub string) ([]RedditPost, error) {
	feedURL := fmt.Sprintf("https://www.reddit.com/r/%s/.rss", sub)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, err
	}

	return parseAtomFeed(sub, body)
}

// ── Atom XML types (Reddit serves Atom 1.0) ────────────────────────────

type atomFeed struct {
	XMLName xml.Name    `xml:"feed"`
	Entries []atomEntry `xml:"entry"`
}

type atomEntry struct {
	ID      string     `xml:"id"`
	Title   string     `xml:"title"`
	Links   []atomLink `xml:"link"`
	Author  atomAuthor `xml:"author"`
	Content string     `xml:"content"`
	Updated string     `xml:"updated"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
}

type atomAuthor struct {
	Name string `xml:"name"`
}

func parseAtomFeed(subreddit string, raw []byte) ([]RedditPost, error) {
	var feed atomFeed
	if err := xml.Unmarshal(raw, &feed); err != nil {
		return nil, fmt.Errorf("parse atom: %w", err)
	}

	posts := make([]RedditPost, 0, len(feed.Entries))
	for _, e := range feed.Entries {
		updated, _ := time.Parse(time.RFC3339, e.Updated)
		posts = append(posts, RedditPost{
			Title:     strings.TrimSpace(e.Title),
			Body:      stripHTML(e.Content),
			URL:       postLink(e),
			Author:    e.Author.Name,
			Subreddit: subreddit,
			UpdatedAt: updated,
		})
	}
	return posts, nil
}

func postLink(e atomEntry) string {
	for _, l := range e.Links {
		if l.Rel == "alternate" {
			return l.Href
		}
	}
	for _, l := range e.Links {
		if l.Href != "" {
			return l.Href
		}
	}
	return e.ID
}

// stripHTML removes HTML tags from content (Reddit wraps Atom content in HTML).
func stripHTML(s string) string {
	var out strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			out.WriteRune(r)
		}
	}
	return strings.TrimSpace(out.String())
}
