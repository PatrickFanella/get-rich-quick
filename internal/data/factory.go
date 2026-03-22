package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/config"
	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
)

const (
	cacheProviderStockChain   = "stock-chain"
	cacheProviderCryptoChain  = "crypto-chain"
	cacheDataTypeOHLCV        = "ohlcv"
	cacheDataTypeFundamentals = "fundamentals"
	cacheDataTypeNews         = "news"
)

var ErrUnsupportedMarketType = errors.New("data: unsupported market type")

type (
	polygonProviderFactoryType      func(apiKey string, logger *slog.Logger) DataProvider
	alphaVantageProviderFactoryType func(apiKey string, rateLimitPerMinute int, logger *slog.Logger) DataProvider
	loggerProviderFactoryType       func(logger *slog.Logger) DataProvider
)

var (
	polygonProviderFactory      polygonProviderFactoryType
	alphaVantageProviderFactory alphaVantageProviderFactoryType
	yahooProviderFactory        loggerProviderFactoryType
	binanceProviderFactory      loggerProviderFactoryType
)

// RegisterPolygonProviderFactory registers the Polygon provider constructor used by NewDataService.
func RegisterPolygonProviderFactory(factory polygonProviderFactoryType) {
	polygonProviderFactory = factory
}

// RegisterAlphaVantageProviderFactory registers the Alpha Vantage provider constructor used by NewDataService.
func RegisterAlphaVantageProviderFactory(factory alphaVantageProviderFactoryType) {
	alphaVantageProviderFactory = factory
}

// RegisterYahooProviderFactory registers the Yahoo provider constructor used by NewDataService.
func RegisterYahooProviderFactory(factory loggerProviderFactoryType) {
	yahooProviderFactory = factory
}

// RegisterBinanceProviderFactory registers the Binance provider constructor used by NewDataService.
func RegisterBinanceProviderFactory(factory loggerProviderFactoryType) {
	binanceProviderFactory = factory
}

// DataService wraps market-data provider chains with cache lookups and writes.
type DataService struct {
	stockChain  DataProvider
	cryptoChain DataProvider
	cacheRepo   repository.MarketDataCacheRepository
	logger      *slog.Logger
	now         func() time.Time
}

// NewDataService constructs provider chains for each supported market type and
// wraps them with cache access.
func NewDataService(cfg config.Config, cacheRepo repository.MarketDataCacheRepository, logger *slog.Logger) *DataService {
	if logger == nil {
		logger = slog.Default()
	}

	stockProviders := make([]DataProvider, 0, 3)
	if apiKey := strings.TrimSpace(cfg.DataProviders.Polygon.APIKey); apiKey != "" && polygonProviderFactory != nil {
		stockProviders = append(stockProviders, polygonProviderFactory(apiKey, logger))
	}
	if apiKey := strings.TrimSpace(cfg.DataProviders.AlphaVantage.APIKey); apiKey != "" && alphaVantageProviderFactory != nil {
		stockProviders = append(stockProviders, alphaVantageProviderFactory(apiKey, cfg.DataProviders.AlphaVantage.RateLimitPerMinute, logger))
	}
	if yahooProviderFactory != nil {
		stockProviders = append(stockProviders, yahooProviderFactory(logger))
	}

	cryptoProviders := make([]DataProvider, 0, 1)
	if binanceProviderFactory != nil {
		cryptoProviders = append(cryptoProviders, binanceProviderFactory(logger))
	}

	return &DataService{
		stockChain:  NewProviderChain(logger, stockProviders...),
		cryptoChain: NewProviderChain(logger, cryptoProviders...),
		cacheRepo:   cacheRepo,
		logger:      logger,
		now:         time.Now,
	}
}

// GetOHLCV returns OHLCV data using the market-type chain and caches results by query.
func (s *DataService) GetOHLCV(ctx context.Context, marketType domain.MarketType, ticker string, timeframe Timeframe, from, to time.Time) ([]domain.OHLCV, error) {
	fromUTC := from.UTC()
	toUTC := to.UTC()

	providerName, chain, err := s.resolveChain(marketType)
	if err != nil {
		return nil, err
	}

	key := repository.MarketDataCacheKey{
		Ticker:    ticker,
		Provider:  providerName,
		DataType:  cacheDataTypeOHLCV,
		Timeframe: timeframe.String(),
		DateFrom:  &fromUTC,
		DateTo:    &toUTC,
	}

	if cached, ok := s.loadCachedOHLCV(ctx, key); ok {
		return cached, nil
	}

	bars, err := chain.GetOHLCV(ctx, ticker, timeframe, from, to)
	if err != nil {
		return nil, err
	}

	s.storeCached(ctx, key, bars, ttlForOHLCV(timeframe))

	return bars, nil
}

// GetFundamentals returns fundamentals using the market-type chain and caches results.
func (s *DataService) GetFundamentals(ctx context.Context, marketType domain.MarketType, ticker string) (Fundamentals, error) {
	providerName, chain, err := s.resolveChain(marketType)
	if err != nil {
		return Fundamentals{}, err
	}

	key := repository.MarketDataCacheKey{
		Ticker:   ticker,
		Provider: providerName,
		DataType: cacheDataTypeFundamentals,
	}

	if cached, ok := s.loadCachedFundamentals(ctx, key); ok {
		return cached, nil
	}

	fundamentals, err := chain.GetFundamentals(ctx, ticker)
	if err != nil {
		return Fundamentals{}, err
	}

	s.storeCached(ctx, key, fundamentals, 6*time.Hour)

	return fundamentals, nil
}

// GetNews returns news using the market-type chain and caches results by query window.
func (s *DataService) GetNews(ctx context.Context, marketType domain.MarketType, ticker string, from, to time.Time) ([]NewsArticle, error) {
	fromUTC := from.UTC()
	toUTC := to.UTC()

	providerName, chain, err := s.resolveChain(marketType)
	if err != nil {
		return nil, err
	}

	key := repository.MarketDataCacheKey{
		Ticker:   ticker,
		Provider: providerName,
		DataType: cacheDataTypeNews,
		DateFrom: &fromUTC,
		DateTo:   &toUTC,
	}

	if cached, ok := s.loadCachedNews(ctx, key); ok {
		return cached, nil
	}

	articles, err := chain.GetNews(ctx, ticker, from, to)
	if err != nil {
		return nil, err
	}

	s.storeCached(ctx, key, articles, 30*time.Minute)

	return articles, nil
}

func (s *DataService) resolveChain(marketType domain.MarketType) (string, DataProvider, error) {
	switch normalizeMarketType(marketType) {
	case domain.MarketTypeStock:
		return cacheProviderStockChain, s.stockChain, nil
	case domain.MarketTypeCrypto:
		return cacheProviderCryptoChain, s.cryptoChain, nil
	default:
		return "", nil, fmt.Errorf("%w: %s", ErrUnsupportedMarketType, marketType)
	}
}

func (s *DataService) loadCachedOHLCV(ctx context.Context, key repository.MarketDataCacheKey) ([]domain.OHLCV, bool) {
	var bars []domain.OHLCV
	return bars, s.loadCached(ctx, key, &bars)
}

func (s *DataService) loadCachedFundamentals(ctx context.Context, key repository.MarketDataCacheKey) (Fundamentals, bool) {
	var fundamentals Fundamentals
	return fundamentals, s.loadCached(ctx, key, &fundamentals)
}

func (s *DataService) loadCachedNews(ctx context.Context, key repository.MarketDataCacheKey) ([]NewsArticle, bool) {
	var news []NewsArticle
	return news, s.loadCached(ctx, key, &news)
}

func (s *DataService) loadCached(ctx context.Context, key repository.MarketDataCacheKey, dest any) bool {
	if s == nil || s.cacheRepo == nil {
		return false
	}

	entry, err := s.cacheRepo.Get(ctx, key)
	if err != nil || entry == nil {
		return false
	}

	if err := json.Unmarshal(entry.Data, dest); err != nil {
		s.logger.Warn("failed to decode cached market data, refreshing",
			slog.String("ticker", key.Ticker),
			slog.String("provider", key.Provider),
			slog.String("data_type", key.DataType),
			slog.Any("error", err),
		)
		return false
	}

	return true
}

func (s *DataService) storeCached(ctx context.Context, key repository.MarketDataCacheKey, value any, ttl time.Duration) {
	if s == nil || s.cacheRepo == nil {
		return
	}

	payload, err := json.Marshal(value)
	if err != nil {
		s.logger.Warn("failed to encode market data for cache",
			slog.String("ticker", key.Ticker),
			slog.String("provider", key.Provider),
			slog.String("data_type", key.DataType),
			slog.Any("error", err),
		)
		return
	}

	fetchedAt := s.currentTime().UTC()
	if err := s.cacheRepo.Set(ctx, &domain.MarketData{
		Ticker:    key.Ticker,
		Provider:  key.Provider,
		DataType:  key.DataType,
		Timeframe: key.Timeframe,
		DateFrom:  key.DateFrom,
		DateTo:    key.DateTo,
		Data:      payload,
		FetchedAt: fetchedAt,
		ExpiresAt: fetchedAt.Add(ttl),
	}); err != nil {
		s.logger.Warn("failed to store market data in cache",
			slog.String("ticker", key.Ticker),
			slog.String("provider", key.Provider),
			slog.String("data_type", key.DataType),
			slog.Any("error", err),
		)
	}
}

func (s *DataService) currentTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now()
	}

	return s.now()
}

func ttlForOHLCV(timeframe Timeframe) time.Duration {
	if timeframe == Timeframe1d {
		return 24 * time.Hour
	}

	return 5 * time.Minute
}

func normalizeMarketType(marketType domain.MarketType) domain.MarketType {
	return domain.MarketType(strings.ToLower(strings.TrimSpace(marketType.String())))
}
