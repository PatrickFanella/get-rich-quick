package llm

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrBudgetExhausted is returned when daily request or token budget is spent.
// RetryProvider treats this as non-retryable (unknown error type, no statusCoder).
var ErrBudgetExhausted = errors.New("llm: daily budget exhausted")

// Budget tracks daily request and token usage with thread-safe counters.
// Zero limits mean unlimited. Call Reset() to clear counters (e.g. at UTC midnight).
type Budget struct {
	mu sync.Mutex

	maxRequestsPerDay int
	maxTokensPerDay   int

	requests         int
	promptTokens     int
	completionTokens int
	resetAt          time.Time
}

// NewBudget creates a Budget with the given daily limits.
// Zero values for either limit mean that dimension is unlimited.
func NewBudget(maxRequestsPerDay, maxTokensPerDay int) *Budget {
	return &Budget{
		maxRequestsPerDay: maxRequestsPerDay,
		maxTokensPerDay:   maxTokensPerDay,
		resetAt:           nextUTCMidnight(),
	}
}

// Allow returns true if a new request is within budget.
// It auto-resets counters when UTC midnight has passed.
func (b *Budget) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.maybeReset()

	if b.maxRequestsPerDay > 0 && b.requests >= b.maxRequestsPerDay {
		return false
	}
	if b.maxTokensPerDay > 0 && (b.promptTokens+b.completionTokens) >= b.maxTokensPerDay {
		return false
	}
	return true
}

// Record tracks token usage for a completed request.
func (b *Budget) Record(promptTokens, completionTokens int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.maybeReset()

	b.requests++
	b.promptTokens += promptTokens
	b.completionTokens += completionTokens
}

// Reset clears all counters and sets the next reset time.
func (b *Budget) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.requests = 0
	b.promptTokens = 0
	b.completionTokens = 0
	b.resetAt = nextUTCMidnight()
}

// Stats returns current budget usage (for observability).
func (b *Budget) Stats() BudgetStats {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.maybeReset()

	return BudgetStats{
		Requests:            b.requests,
		MaxRequestsPerDay:   b.maxRequestsPerDay,
		PromptTokens:        b.promptTokens,
		CompletionTokens:    b.completionTokens,
		TotalTokens:         b.promptTokens + b.completionTokens,
		MaxTokensPerDay:     b.maxTokensPerDay,
		ResetAt:             b.resetAt,
	}
}

// BudgetStats holds a snapshot of budget usage.
type BudgetStats struct {
	Requests          int
	MaxRequestsPerDay int
	PromptTokens      int
	CompletionTokens  int
	TotalTokens       int
	MaxTokensPerDay   int
	ResetAt           time.Time
}

// maybeReset auto-resets counters when past the reset time. Must hold mu.
func (b *Budget) maybeReset() {
	if time.Now().UTC().After(b.resetAt) {
		b.requests = 0
		b.promptTokens = 0
		b.completionTokens = 0
		b.resetAt = nextUTCMidnight()
	}
}

func nextUTCMidnight() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, time.UTC)
}

// BudgetGuardProvider wraps a Provider, rejecting calls when budget is exhausted.
// On success it records token usage back to the budget.
type BudgetGuardProvider struct {
	inner  Provider
	budget *Budget
}

// NewBudgetGuardProvider wraps inner with budget enforcement.
func NewBudgetGuardProvider(inner Provider, budget *Budget) *BudgetGuardProvider {
	return &BudgetGuardProvider{inner: inner, budget: budget}
}

// Complete checks budget before delegating. Records usage after success.
func (b *BudgetGuardProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !b.budget.Allow() {
		return nil, ErrBudgetExhausted
	}

	resp, err := b.inner.Complete(ctx, req)
	if err != nil {
		return nil, err
	}

	b.budget.Record(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	return resp, nil
}
