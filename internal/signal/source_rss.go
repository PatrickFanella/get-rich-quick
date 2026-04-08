package signal

import (
	"context"
	"log/slog"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/data/rss"
)

// DefaultRSSFeedURLs returns the default financial/general news RSS feed URLs
// used for signal monitoring.
func DefaultRSSFeedURLs() []string {
	return []string{
		"https://feeds.reuters.com/reuters/topNews",
		"https://feeds.apnews.com/rss/apf-topnews",
		"https://feeds.marketwatch.com/marketwatch/topstories/",
		"https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=100003114",
	}
}

// RSSSource is a SignalSource that polls RSS feeds and emits new articles as
// RawSignalEvents. Deduplication is handled by the underlying rss.Aggregator
// (24-hour seen-URL cache).
type RSSSource struct {
	agg      *rss.Aggregator
	interval time.Duration
	logger   *slog.Logger
}

// NewRSSSource creates an RSS signal source for the given feed URLs.
// If interval is zero it defaults to 60 seconds.
func NewRSSSource(feedURLs []string, interval time.Duration, logger *slog.Logger) *RSSSource {
	if interval == 0 {
		interval = 60 * time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	feeds := make([]rss.Feed, len(feedURLs))
	for i, u := range feedURLs {
		feeds[i] = rss.Feed{Name: u, URL: u}
	}
	return &RSSSource{
		agg:      rss.NewAggregator(feeds, logger),
		interval: interval,
		logger:   logger,
	}
}

func (r *RSSSource) Name() string { return "rss" }

// Start polls all configured feeds on the given interval, emitting one
// RawSignalEvent per new article. The channel is closed when ctx is cancelled.
func (r *RSSSource) Start(ctx context.Context) (<-chan RawSignalEvent, error) {
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
				for _, a := range r.agg.Fetch(ctx) {
					evt := RawSignalEvent{
						Source:     "rss:" + a.Source,
						Title:      a.Title,
						Body:       a.Description,
						URL:        a.Link,
						Metadata:   map[string]any{"guid": a.GUID},
						ReceivedAt: time.Now(),
					}
					select {
					case ch <- evt:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	return ch, nil
}
