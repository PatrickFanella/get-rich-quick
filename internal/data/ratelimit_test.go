package data_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/data"
)

func TestRateLimiterTryAcquireRespectsRate(t *testing.T) {
	t.Parallel()

	limiter := data.NewRateLimiter(2, 100*time.Millisecond)

	if !limiter.TryAcquire() {
		t.Fatal("TryAcquire() = false, want true for first token")
	}
	if !limiter.TryAcquire() {
		t.Fatal("TryAcquire() = false, want true for second token")
	}
	if limiter.TryAcquire() {
		t.Fatal("TryAcquire() = true, want false when bucket is exhausted")
	}

	time.Sleep(110 * time.Millisecond)

	if !limiter.TryAcquire() {
		t.Fatal("TryAcquire() = false, want true after refill interval")
	}
}

func TestRateLimiterWaitBlocksUntilTokenAvailable(t *testing.T) {
	t.Parallel()

	const refillInterval = 80 * time.Millisecond

	limiter := data.NewRateLimiter(1, refillInterval)
	if !limiter.TryAcquire() {
		t.Fatal("TryAcquire() = false, want true for initial token")
	}

	start := time.Now()
	done := make(chan error, 1)
	go func() {
		done <- limiter.Wait(context.Background())
	}()

	select {
	case err := <-done:
		t.Fatalf("Wait() returned early with err = %v, want it to block", err)
	case <-time.After(30 * time.Millisecond):
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Wait() error = %v, want nil", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Wait() did not return after token should have refilled")
	}

	if elapsed := time.Since(start); elapsed < refillInterval-20*time.Millisecond {
		t.Fatalf("Wait() blocked for %v, want at least %v", elapsed, refillInterval-20*time.Millisecond)
	}
}

func TestRateLimiterWaitHonorsContextCancellation(t *testing.T) {
	t.Parallel()

	limiter := data.NewRateLimiter(1, 200*time.Millisecond)
	if !limiter.TryAcquire() {
		t.Fatal("TryAcquire() = false, want true for initial token")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := limiter.Wait(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait() error = %v, want context deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 150*time.Millisecond {
		t.Fatalf("Wait() returned after %v, want prompt cancellation", elapsed)
	}
}
