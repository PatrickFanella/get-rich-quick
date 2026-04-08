package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/PatrickFanella/get-rich-quick/internal/agent"
	"github.com/PatrickFanella/get-rich-quick/internal/config"
	"github.com/PatrickFanella/get-rich-quick/internal/llm"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
	"github.com/PatrickFanella/get-rich-quick/internal/signal"
)

// signalStrategyProvider adapts StrategyRepository to signal.StrategyProvider.
// It lists all active strategies and enriches each with its stored thesis
// watch terms so the SignalHub's WatchIndex stays current.
type signalStrategyProvider struct {
	repo repository.StrategyRepository
}

func (p *signalStrategyProvider) ListActiveWithThesis(ctx context.Context) ([]signal.StrategyWithThesis, error) {
	strategies, err := p.repo.List(ctx, repository.StrategyFilter{Status: "active"}, 500, 0)
	if err != nil {
		return nil, err
	}

	result := make([]signal.StrategyWithThesis, 0, len(strategies))
	for _, s := range strategies {
		sw := signal.StrategyWithThesis{
			ID:     s.ID,
			Ticker: s.Ticker,
		}
		// Best-effort: load the stored thesis to extract watch terms.
		raw, err := p.repo.GetThesisRaw(ctx, s.ID)
		if err == nil && len(raw) > 0 {
			var t agent.Thesis
			if json.Unmarshal(raw, &t) == nil {
				sw.WatchTerms = t.WatchTerms
			}
		}
		result = append(result, sw)
	}
	return result, nil
}

// thesisLoaderRepo adapts StrategyRepository to signal.ThesisLoader.
type thesisLoaderRepo struct {
	repo repository.StrategyRepository
}

func (t *thesisLoaderRepo) GetThesisRaw(ctx context.Context, strategyID uuid.UUID) (json.RawMessage, error) {
	return t.repo.GetThesisRaw(ctx, strategyID)
}

// signalTriggerer adapts a StrategyTrigger (from the scheduler) to
// signal.StrategyTriggerer without an import cycle.
type signalTriggerer interface {
	TriggerStrategy(strategy interface{ GetID() uuid.UUID })
}

// buildSignalInfra constructs the full signal intelligence stack:
// EventStore → WatchIndex → SignalHub (sources + evaluator) → TriggerHandler.
//
// Returns the EventStore and WatchIndex (wired into API deps by the caller),
// plus a shutdown function that must be called on process exit.
//
// The hub is only started when the provided runner is non-nil (i.e., when the
// full strategy runner is wired). In smoke mode, pass nil for runner.
func buildSignalInfra(
	ctx context.Context,
	cfg config.Config,
	strategyRepo repository.StrategyRepository,
	polymarketAccountRepo repository.PolymarketAccountRepository,
	llmProvider llm.Provider,
	runner signal.StrategyTriggerer,
	logger *slog.Logger,
) (store *signal.EventStore, watchIndex *signal.WatchIndex, shutdown func()) {
	store = signal.NewEventStore(200)
	watchIndex = signal.NewWatchIndex()

	if runner == nil {
		// Smoke / test mode: return the empty store and index but don't start the hub.
		return store, watchIndex, func() {}
	}

	// Build the LLM evaluator (optional; nil = urgency-3 fallback for all events).
	var evaluator *signal.Evaluator
	if llmProvider != nil {
		evaluator = signal.NewEvaluator(llmProvider, cfg.LLM.QuickThinkModel, logger)
	}

	// Signal sources — always include RSS and Reddit; conditionally add Polymarket sources.
	var sources []signal.SignalSource
	sources = append(sources,
		signal.NewRSSSource(signal.DefaultRSSFeedURLs(), 60*time.Second, logger),
		signal.NewRedditSource(signal.DefaultSubreddits(), 60*time.Second, logger),
	)

	clobURL := cfg.Brokers.Polymarket.CLOBURL
	if clobURL == "" {
		clobURL = "https://clob.polymarket.com"
	}

	// Polymarket CLOB price/volume monitor (active only when strategies exist).
	sources = append(sources, signal.NewPolymarketSource(signal.PolymarketSourceConfig{
		CLOBURL:               clobURL,
		Interval:              10 * time.Minute,
		PriceMoveThreshold:    0.05,
		VolumeSpikeMultiplier: 3.0,
	}, logger))

	// Whale tracker (active when Polymarket account repo is available).
	if polymarketAccountRepo != nil {
		sources = append(sources, signal.NewWhaleSource(signal.WhaleSourceConfig{
			CLOBURL:      clobURL,
			Interval:     30 * time.Second,
			MinTradeUSDC: 5000,
			MinWinRate:   0.65,
		}, polymarketAccountRepo, logger))
	}

	stratProvider := &signalStrategyProvider{repo: strategyRepo}
	thesisLoader := &thesisLoaderRepo{repo: strategyRepo}

	triggerCh := make(chan signal.TriggerEvent, 64)

	hub := signal.NewSignalHub(sources, evaluator, watchIndex, stratProvider, triggerCh, store, logger)
	handler := signal.NewTriggerHandler(triggerCh, strategyRepo, thesisLoader, runner, store, logger)

	if err := hub.Start(ctx); err != nil {
		logger.Warn("signal hub: failed to start", slog.Any("error", err))
		return store, watchIndex, func() {}
	}
	logger.Info("signal intelligence: hub started", slog.Int("sources", len(sources)))

	handlerCtx, cancelHandler := context.WithCancel(ctx)
	go handler.Run(handlerCtx)

	return store, watchIndex, func() {
		hub.Stop()
		cancelHandler()
	}
}
