package llm_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/PatrickFanella/get-rich-quick/internal/llm"
)

func TestBudget_AllowUnlimited(t *testing.T) {
	t.Parallel()

	b := llm.NewBudget(0, 0) // unlimited
	for range 100 {
		if !b.Allow() {
			t.Fatal("Allow() returned false for unlimited budget")
		}
		b.Record(100, 50)
	}
}

func TestBudget_RequestLimit(t *testing.T) {
	t.Parallel()

	b := llm.NewBudget(3, 0)

	for i := range 3 {
		if !b.Allow() {
			t.Fatalf("Allow() false at request %d, want true", i+1)
		}
		b.Record(10, 5)
	}

	if b.Allow() {
		t.Error("Allow() true after 3/3 requests, want false")
	}
}

func TestBudget_TokenLimit(t *testing.T) {
	t.Parallel()

	b := llm.NewBudget(0, 100) // 100 total tokens

	b.Record(40, 30) // 70 total
	if !b.Allow() {
		t.Fatal("Allow() false at 70/100 tokens")
	}

	b.Record(20, 15) // 105 total → over
	if b.Allow() {
		t.Error("Allow() true at 105/100 tokens, want false")
	}
}

func TestBudget_Reset(t *testing.T) {
	t.Parallel()

	b := llm.NewBudget(1, 0)
	b.Record(10, 5)

	if b.Allow() {
		t.Fatal("Allow() true at 1/1 requests before reset")
	}

	b.Reset()

	if !b.Allow() {
		t.Error("Allow() false after Reset(), want true")
	}
}

func TestBudget_Stats(t *testing.T) {
	t.Parallel()

	b := llm.NewBudget(10, 500)
	b.Record(20, 10)
	b.Record(30, 15)

	s := b.Stats()
	if s.Requests != 2 {
		t.Errorf("Requests = %d, want 2", s.Requests)
	}
	if s.PromptTokens != 50 {
		t.Errorf("PromptTokens = %d, want 50", s.PromptTokens)
	}
	if s.CompletionTokens != 25 {
		t.Errorf("CompletionTokens = %d, want 25", s.CompletionTokens)
	}
	if s.TotalTokens != 75 {
		t.Errorf("TotalTokens = %d, want 75", s.TotalTokens)
	}
	if s.MaxRequestsPerDay != 10 {
		t.Errorf("MaxRequestsPerDay = %d, want 10", s.MaxRequestsPerDay)
	}
	if s.MaxTokensPerDay != 500 {
		t.Errorf("MaxTokensPerDay = %d, want 500", s.MaxTokensPerDay)
	}
}

func TestBudgetGuardProvider_Blocks(t *testing.T) {
	t.Parallel()

	inner := &trackingProvider{
		response: &llm.CompletionResponse{Content: "ok", Usage: llm.CompletionUsage{PromptTokens: 5, CompletionTokens: 3}},
	}
	budget := llm.NewBudget(2, 0)
	guard := llm.NewBudgetGuardProvider(inner, budget)

	// Two calls succeed
	for i := range 2 {
		_, err := guard.Complete(context.Background(), llm.CompletionRequest{})
		if err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
	}

	// Third blocked
	_, err := guard.Complete(context.Background(), llm.CompletionRequest{})
	if !errors.Is(err, llm.ErrBudgetExhausted) {
		t.Errorf("call 3 error = %v, want ErrBudgetExhausted", err)
	}
	if inner.calls.Load() != 2 {
		t.Errorf("inner calls = %d, want 2", inner.calls.Load())
	}
}

func TestBudgetGuardProvider_RecordsUsage(t *testing.T) {
	t.Parallel()

	inner := &trackingProvider{
		response: &llm.CompletionResponse{
			Content: "tracked",
			Usage:   llm.CompletionUsage{PromptTokens: 20, CompletionTokens: 10},
		},
	}
	budget := llm.NewBudget(0, 50) // 50 token limit
	guard := llm.NewBudgetGuardProvider(inner, budget)

	// First call: 30 tokens
	_, err := guard.Complete(context.Background(), llm.CompletionRequest{})
	if err != nil {
		t.Fatalf("call 1: %v", err)
	}

	s := budget.Stats()
	if s.TotalTokens != 30 {
		t.Errorf("after call 1: TotalTokens = %d, want 30", s.TotalTokens)
	}

	// Second call: 60 tokens total → over budget
	_, err = guard.Complete(context.Background(), llm.CompletionRequest{})
	if err != nil {
		t.Fatalf("call 2: %v", err)
	}

	// Third call: blocked (60 ≥ 50)
	_, err = guard.Complete(context.Background(), llm.CompletionRequest{})
	if !errors.Is(err, llm.ErrBudgetExhausted) {
		t.Errorf("call 3 error = %v, want ErrBudgetExhausted", err)
	}
}

func TestBudgetGuardProvider_ConcurrentRequestLimit(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})

	inner := llm.ProviderFunc(func(_ context.Context, _ llm.CompletionRequest) (*llm.CompletionResponse, error) {
		calls.Add(1)
		close(started)
		<-release
		return &llm.CompletionResponse{Content: "ok"}, nil
	})

	budget := llm.NewBudget(1, 0)
	guard := llm.NewBudgetGuardProvider(inner, budget)

	var wg sync.WaitGroup
	wg.Add(2)

	var err1, err2 error
	go func() {
		defer wg.Done()
		_, err1 = guard.Complete(context.Background(), llm.CompletionRequest{})
	}()
	<-started
	go func() {
		defer wg.Done()
		_, err2 = guard.Complete(context.Background(), llm.CompletionRequest{})
	}()

	close(release)
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("inner calls = %d, want 1", calls.Load())
	}
	if (errors.Is(err1, llm.ErrBudgetExhausted) && err2 == nil) || (errors.Is(err2, llm.ErrBudgetExhausted) && err1 == nil) {
		return
	}
	t.Fatalf("expected one success and one ErrBudgetExhausted, got err1=%v err2=%v", err1, err2)
}

func TestErrBudgetExhausted_IsNotRetryable(t *testing.T) {
	t.Parallel()

	// ErrBudgetExhausted is a plain sentinel → no statusCoder interface →
	// RetryProvider.isRetryable returns false. Verify by trying to retry it.
	calls := 0
	failing := llm.ProviderFunc(func(_ context.Context, _ llm.CompletionRequest) (*llm.CompletionResponse, error) {
		calls++
		return nil, llm.ErrBudgetExhausted
	})

	rp := llm.NewRetryProvider(failing, discardLogger(), llm.WithMaxAttempts(3))
	rp.SetTimerFn(immediateTimerFn())

	_, err := rp.Complete(context.Background(), llm.CompletionRequest{})
	if !errors.Is(err, llm.ErrBudgetExhausted) {
		t.Errorf("error = %v, want ErrBudgetExhausted", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (should not retry budget errors)", calls)
	}
}
