package backtest

import (
	"context"
	"sort"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// HistoricalDataProvider implements data.DataProvider using pre-loaded
// historical data and a simulated clock. Only data with timestamps at or
// before the clock's current time is returned, preventing look-ahead bias.
type HistoricalDataProvider struct {
	bars         []domain.OHLCV       // sorted ascending by Timestamp
	news         []data.NewsArticle   // sorted ascending by PublishedAt
	fundamentals *data.Fundamentals   // static; always returned when non-nil
	social       []data.SocialSentiment // sorted ascending by MeasuredAt
	clock        Clock
}

// HistoricalDataOption configures a HistoricalDataProvider during construction.
type HistoricalDataOption func(*HistoricalDataProvider)

// WithBars supplies OHLCV bars to the provider. The bars are sorted by
// timestamp internally; callers do not need to pre-sort.
func WithBars(bars []domain.OHLCV) HistoricalDataOption {
	return func(p *HistoricalDataProvider) {
		sorted := make([]domain.OHLCV, len(bars))
		copy(sorted, bars)
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].Timestamp.Before(sorted[j].Timestamp)
		})
		p.bars = sorted
	}
}

// WithNews supplies news articles to the provider. Articles are sorted by
// PublishedAt internally.
func WithNews(news []data.NewsArticle) HistoricalDataOption {
	return func(p *HistoricalDataProvider) {
		sorted := make([]data.NewsArticle, len(news))
		copy(sorted, news)
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].PublishedAt.Before(sorted[j].PublishedAt)
		})
		p.news = sorted
	}
}

// WithFundamentals supplies static fundamental data to the provider.
func WithFundamentals(f *data.Fundamentals) HistoricalDataOption {
	return func(p *HistoricalDataProvider) {
		p.fundamentals = f
	}
}

// WithSocialSentiment supplies social-sentiment snapshots to the provider.
// Snapshots are sorted by MeasuredAt internally.
func WithSocialSentiment(social []data.SocialSentiment) HistoricalDataOption {
	return func(p *HistoricalDataProvider) {
		sorted := make([]data.SocialSentiment, len(social))
		copy(sorted, social)
		sort.SliceStable(sorted, func(i, j int) bool {
			return sorted[i].MeasuredAt.Before(sorted[j].MeasuredAt)
		})
		p.social = sorted
	}
}

// NewHistoricalDataProvider creates a provider backed by historical data. The
// clock determines the upper bound for accessible data, ensuring no future data
// is leaked. Use With* options to supply bars, news, fundamentals, and social
// sentiment.
func NewHistoricalDataProvider(clock Clock, opts ...HistoricalDataOption) *HistoricalDataProvider {
	p := &HistoricalDataProvider{clock: clock}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// Compile-time check: HistoricalDataProvider implements data.DataProvider.
var _ data.DataProvider = (*HistoricalDataProvider)(nil)

// GetOHLCV returns bars within [from, to] whose timestamps do not exceed the
// current simulated time. The timeframe parameter is accepted for interface
// compliance but does not filter results; callers should pre-load bars of the
// desired timeframe.
func (p *HistoricalDataProvider) GetOHLCV(_ context.Context, _ string, _ data.Timeframe, from, to time.Time) ([]domain.OHLCV, error) {
	now := p.clock.Now()
	var result []domain.OHLCV
	for _, bar := range p.bars {
		if bar.Timestamp.After(now) {
			break
		}
		if !bar.Timestamp.Before(from) && !bar.Timestamp.After(to) {
			result = append(result, bar)
		}
	}
	return result, nil
}

// GetFundamentals returns the pre-loaded fundamentals. If none were supplied,
// data.ErrNotImplemented is returned so that a provider chain can fall through.
func (p *HistoricalDataProvider) GetFundamentals(_ context.Context, _ string) (data.Fundamentals, error) {
	if p.fundamentals == nil {
		return data.Fundamentals{}, data.ErrNotImplemented
	}
	return *p.fundamentals, nil
}

// GetNews returns articles within [from, to] whose PublishedAt does not exceed
// the current simulated time.
func (p *HistoricalDataProvider) GetNews(_ context.Context, _ string, from, to time.Time) ([]data.NewsArticle, error) {
	now := p.clock.Now()
	var result []data.NewsArticle
	for _, article := range p.news {
		if article.PublishedAt.After(now) {
			break
		}
		if !article.PublishedAt.Before(from) && !article.PublishedAt.After(to) {
			result = append(result, article)
		}
	}
	return result, nil
}

// GetSocialSentiment returns sentiment snapshots within [from, to] whose
// MeasuredAt does not exceed the current simulated time.
func (p *HistoricalDataProvider) GetSocialSentiment(_ context.Context, _ string, from, to time.Time) ([]data.SocialSentiment, error) {
	now := p.clock.Now()
	var result []data.SocialSentiment
	for _, s := range p.social {
		if s.MeasuredAt.After(now) {
			break
		}
		if !s.MeasuredAt.Before(from) && !s.MeasuredAt.After(to) {
			result = append(result, s)
		}
	}
	return result, nil
}
