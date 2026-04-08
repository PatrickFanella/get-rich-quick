package signal

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

// DefaultSubreddits returns the default subreddits to monitor for signal events.
func DefaultSubreddits() []string {
	return []string{
		"politics",
		"worldnews",
		"cryptocurrency",
		"polymarket",
		"wallstreetbets",
	}
}

// RedditSource is a SignalSource that polls Reddit Atom feeds and emits new
// posts as RawSignalEvents. Deduplication uses a 24-hour in-memory seen set.
type RedditSource struct {
	subreddits []string
	interval   time.Duration
	client     *http.Client
	logger     *slog.Logger

	mu   sync.Mutex
	seen map[string]time.Time // post URL → first seen
}

// NewRedditSource creates a Reddit signal source for the given subreddits.
// If interval is zero it defaults to 60 seconds.
func NewRedditSource(subreddits []string, interval time.Duration, logger *slog.Logger) *RedditSource {
	if interval == 0 {
		interval = 60 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &RedditSource{
		subreddits: subreddits,
		interval:   interval,
		client:     &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
		seen:       make(map[string]time.Time),
	}
}

func (r *RedditSource) Name() string { return "reddit" }

// Start polls each subreddit's RSS feed on the given interval, emitting one
// RawSignalEvent per new post. The channel is closed when ctx is cancelled.
func (r *RedditSource) Start(ctx context.Context) (<-chan RawSignalEvent, error) {
	ch := make(chan RawSignalEvent, 64)
	go func() {
		defer close(ch)
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				posts := r.fetchAll(ctx)
				for _, p := range posts {
					select {
					case ch <- p:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch, nil
}

func (r *RedditSource) fetchAll(ctx context.Context) []RawSignalEvent {
	var (
		mu     sync.Mutex
		events []RawSignalEvent
		wg     sync.WaitGroup
	)
	for _, sub := range r.subreddits {
		wg.Add(1)
		go func(sub string) {
			defer wg.Done()
			got, err := r.fetchSubreddit(ctx, sub)
			if err != nil {
				r.logger.Warn("reddit: fetch failed",
					slog.String("subreddit", sub),
					slog.Any("error", err),
				)
				return
			}
			mu.Lock()
			events = append(events, got...)
			mu.Unlock()
		}(sub)
	}
	wg.Wait()

	// Deduplicate and prune seen cache.
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)
	for k, t := range r.seen {
		if t.Before(cutoff) {
			delete(r.seen, k)
		}
	}

	var fresh []RawSignalEvent
	for _, e := range events {
		if _, exists := r.seen[e.URL]; exists {
			continue
		}
		r.seen[e.URL] = now
		fresh = append(fresh, e)
	}
	return fresh
}

func (r *RedditSource) fetchSubreddit(ctx context.Context, sub string) ([]RawSignalEvent, error) {
	url := fmt.Sprintf("https://www.reddit.com/r/%s/.rss", sub)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	// Reddit requires a descriptive User-Agent.
	req.Header.Set("User-Agent", "get-rich-quick/1.0 signal-monitor")

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}

	return parseRedditAtom(sub, body)
}

// Reddit serves Atom 1.0 feeds; we parse only the fields we need.
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

func parseRedditAtom(subreddit string, data []byte) ([]RawSignalEvent, error) {
	var feed atomFeed
	if err := xml.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parse atom: %w", err)
	}

	now := time.Now()
	var events []RawSignalEvent
	for _, e := range feed.Entries {
		link := postLink(e)
		events = append(events, RawSignalEvent{
			Source: "reddit",
			Title:  strings.TrimSpace(e.Title),
			Body:   stripAtomHTML(e.Content),
			URL:    link,
			Metadata: map[string]any{
				"subreddit": subreddit,
				"author":    e.Author.Name,
			},
			ReceivedAt: now,
		})
	}
	return events, nil
}

// postLink returns the alternate (post) link from an Atom entry.
func postLink(e atomEntry) string {
	for _, l := range e.Links {
		if l.Rel == "alternate" {
			return l.Href
		}
	}
	// Fallback: first non-empty href.
	for _, l := range e.Links {
		if l.Href != "" {
			return l.Href
		}
	}
	return e.ID
}

// stripAtomHTML strips HTML tags from Atom content (Reddit wraps content in HTML).
func stripAtomHTML(s string) string {
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
