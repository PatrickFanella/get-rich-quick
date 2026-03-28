package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
)

func TestSmokeStrategyRunnerFindRunPagesThroughHistory(t *testing.T) {
	t.Parallel()

	runID := uuid.New()
	repo := &pagingRunRepo{
		items: make([]domain.PipelineRun, 0, 105),
	}
	for i := 0; i < 104; i++ {
		repo.items = append(repo.items, domain.PipelineRun{ID: uuid.New()})
	}
	repo.items = append(repo.items, domain.PipelineRun{
		ID:        runID,
		TradeDate: time.Now().UTC().Truncate(24 * time.Hour),
	})

	runner := &smokeStrategyRunner{runRepo: repo}

	run, err := runner.findRun(context.Background(), runID)
	if err != nil {
		t.Fatalf("findRun() error = %v", err)
	}
	if run.ID != runID {
		t.Fatalf("findRun() id = %s, want %s", run.ID, runID)
	}
	if repo.listCalls < 2 {
		t.Fatalf("expected paged lookup to call List multiple times, got %d", repo.listCalls)
	}
}

type pagingRunRepo struct {
	items     []domain.PipelineRun
	listCalls int
}

func (p *pagingRunRepo) Create(context.Context, *domain.PipelineRun) error { return nil }

func (p *pagingRunRepo) Get(_ context.Context, _ uuid.UUID, _ time.Time) (*domain.PipelineRun, error) {
	return nil, fmt.Errorf("run: %w", repository.ErrNotFound)
}

func (p *pagingRunRepo) List(_ context.Context, _ repository.PipelineRunFilter, limit, offset int) ([]domain.PipelineRun, error) {
	p.listCalls++
	if offset >= len(p.items) {
		return nil, nil
	}
	end := offset + limit
	if end > len(p.items) {
		end = len(p.items)
	}
	return append([]domain.PipelineRun(nil), p.items[offset:end]...), nil
}

func (p *pagingRunRepo) UpdateStatus(context.Context, uuid.UUID, time.Time, repository.PipelineRunStatusUpdate) error {
	return nil
}
