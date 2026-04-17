package llm

import (
	"context"
	"log/slog"
	"time"
)

// ChainOption configures the provider chain built by NewProviderChain.
type ChainOption func(*chainConfig)

type chainConfig struct {
	fallback    Provider
	maxAttempts int
	baseDelay   time.Duration
	throttle    int
	cache       ResponseCache
	budget      *Budget
	callTimeout time.Duration
	metrics     chainMetrics
}

type chainMetrics struct {
	fallback FallbackMetrics
	cache    CacheMetrics
	retry    RetryMetrics
	budget   BudgetMetrics
}

// WithFallback adds a secondary provider tried when primary fails.
func WithFallback(p Provider) ChainOption {
	return func(c *chainConfig) { c.fallback = p }
}

// WithRetry sets the max retry attempts (including initial call).
// Values < 1 are ignored.
func WithRetry(maxAttempts int) ChainOption {
	return func(c *chainConfig) {
		if maxAttempts > 0 {
			c.maxAttempts = maxAttempts
		}
	}
}

// WithRetryBaseDelay sets the base delay for exponential backoff.
func WithRetryBaseDelay(d time.Duration) ChainOption {
	return func(c *chainConfig) {
		if d > 0 {
			c.baseDelay = d
		}
	}
}

// WithThrottle sets the max concurrent calls. Values < 1 are clamped to 1.
func WithThrottle(n int) ChainOption {
	return func(c *chainConfig) {
		if n < 1 {
			n = 1
		}
		c.throttle = n
	}
}

// WithCache enables response caching using the given cache store.
func WithCache(cache ResponseCache) ChainOption {
	return func(c *chainConfig) { c.cache = cache }
}

// WithBudget attaches a budget guard that rejects calls when daily limits are hit.
func WithBudget(b *Budget) ChainOption {
	return func(c *chainConfig) { c.budget = b }
}

// WithCallTimeout wraps each Complete call with a per-call context timeout.
// Zero or negative values disable per-call timeout.
func WithCallTimeout(d time.Duration) ChainOption {
	return func(c *chainConfig) { c.callTimeout = d }
}

// WithChainFallbackMetrics attaches metrics to the fallback layer.
func WithChainFallbackMetrics(m FallbackMetrics) ChainOption {
	return func(c *chainConfig) { c.metrics.fallback = m }
}

// WithChainCacheMetrics attaches metrics to the cache layer.
func WithChainCacheMetrics(m CacheMetrics) ChainOption {
	return func(c *chainConfig) { c.metrics.cache = m }
}

// WithChainRetryMetrics attaches metrics to the retry layer.
func WithChainRetryMetrics(m RetryMetrics) ChainOption {
	return func(c *chainConfig) { c.metrics.retry = m }
}

// WithChainBudgetMetrics attaches metrics to the budget guard layer.
func WithChainBudgetMetrics(m BudgetMetrics) ChainOption {
	return func(c *chainConfig) { c.metrics.budget = m }
}

// NewProviderChain composes a resilient provider from existing primitives.
//
// Chain order (outermost → innermost):
//
//	budget guard → timeout → throttle → retry → fallback → cache → raw provider
//
// Each layer is optional; only layers whose options are provided are added.
// The resulting Provider delegates to the composed chain.
func NewProviderChain(primary Provider, logger *slog.Logger, opts ...ChainOption) Provider {
	if logger == nil {
		logger = slog.Default()
	}

	cfg := chainConfig{
		maxAttempts: 0, // 0 = no retry layer
		throttle:    0, // 0 = no throttle layer
	}
	for _, o := range opts {
		o(&cfg)
	}

	// Build inside-out: start with primary, wrap outward.
	p := primary

	// Layer 1 (innermost): cache
	if cfg.cache != nil {
		cp, err := NewCacheProvider(p, cfg.cache, defaultCacheVersion)
		if err == nil {
			if cfg.metrics.cache != nil {
				cp = cp.WithCacheMetrics(cfg.metrics.cache)
			}
			p = cp
		} else {
			logger.Warn("llm: chain: failed to create cache layer, skipping", slog.Any("error", err))
		}
	}

	// Layer 2: fallback
	if cfg.fallback != nil {
		fp, err := NewFallbackProvider(p, cfg.fallback, logger)
		if err == nil {
			if cfg.metrics.fallback != nil {
				fp = fp.WithMetrics(cfg.metrics.fallback)
			}
			p = fp
		} else {
			logger.Warn("llm: chain: failed to create fallback layer, skipping", slog.Any("error", err))
		}
	}

	// Layer 3: retry
	if cfg.maxAttempts > 1 {
		retryOpts := []RetryOption{WithMaxAttempts(cfg.maxAttempts)}
		if cfg.baseDelay > 0 {
			retryOpts = append(retryOpts, WithBaseDelay(cfg.baseDelay))
		}
		rp := NewRetryProvider(p, logger, retryOpts...)
		if cfg.metrics.retry != nil {
			rp = rp.WithRetryMetrics(cfg.metrics.retry)
		}
		p = rp
	}

	// Layer 4: throttle
	if cfg.throttle > 0 {
		p = NewThrottledProvider(p, cfg.throttle)
	}

	// Layer 5: per-call timeout
	if cfg.callTimeout > 0 {
		p = &timeoutProvider{inner: p, timeout: cfg.callTimeout}
	}

	// Layer 6 (outermost): budget guard
	if cfg.budget != nil {
		bg := NewBudgetGuardProvider(p, cfg.budget)
		if cfg.metrics.budget != nil {
			bg = bg.WithBudgetMetrics(cfg.metrics.budget)
		}
		p = bg
	}

	return p
}

// timeoutProvider wraps each Complete call with a per-call context timeout.
type timeoutProvider struct {
	inner   Provider
	timeout time.Duration
}

func (t *timeoutProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()
	return t.inner.Complete(ctx, req)
}
