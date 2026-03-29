package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
	"github.com/PatrickFanella/get-rich-quick/internal/repository"
)

// TestIntegration_OrderLifecycle_SubmitFillPositionUpdate validates the
// end-to-end flow: submit order → fill order → create trade → position
// update with realized P&L.
func TestIntegration_OrderLifecycle_SubmitFillPositionUpdate(t *testing.T) {
	db := newTestDB(t)
	r := newRepos(db)
	ctx := context.Background()

	// 1. Create strategy and position.
	strategy := createStrategy(t, ctx, r.Strategy, "AAPL Momentum", "AAPL")
	position := createPosition(t, ctx, r.Position, strategy.ID, "AAPL", domain.PositionSideLong, 0, 0)

	// 2. Submit a buy order (pending → submitted).
	runID := uuid.New()
	order := createOrder(t, ctx, r.Order, strategy.ID, &runID, "AAPL", domain.OrderSideBuy, domain.OrderTypeLimit, 10)
	if order.Status != domain.OrderStatusPending {
		t.Fatalf("expected initial status pending, got %q", order.Status)
	}

	submittedAt := time.Now().UTC().Truncate(time.Microsecond)
	order.Status = domain.OrderStatusSubmitted
	order.SubmittedAt = &submittedAt
	if err := r.Order.Update(ctx, order); err != nil {
		t.Fatalf("Update() to submitted: %v", err)
	}

	got, err := r.Order.Get(ctx, order.ID)
	if err != nil {
		t.Fatalf("Get() after submit: %v", err)
	}
	if got.Status != domain.OrderStatusSubmitted {
		t.Fatalf("expected submitted, got %q", got.Status)
	}
	if got.SubmittedAt == nil || !got.SubmittedAt.Equal(submittedAt) {
		t.Fatalf("expected SubmittedAt %v, got %v", submittedAt, got.SubmittedAt)
	}

	// 3. Partially fill the order (submitted → partial).
	order.Status = domain.OrderStatusPartial
	order.FilledQuantity = 5
	filledAvgPrice := 185.50
	order.FilledAvgPrice = &filledAvgPrice
	if err := r.Order.Update(ctx, order); err != nil {
		t.Fatalf("Update() to partial: %v", err)
	}

	// Record first partial fill trade.
	trade1 := &domain.Trade{
		OrderID:    &order.ID,
		PositionID: &position.ID,
		Ticker:     "AAPL",
		Side:       domain.OrderSideBuy,
		Quantity:   5,
		Price:      185.50,
		Fee:        0.25,
		ExecutedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := r.Trade.Create(ctx, trade1); err != nil {
		t.Fatalf("Create() trade1: %v", err)
	}
	if trade1.ID == uuid.Nil {
		t.Fatal("expected trade1 ID to be populated")
	}

	// 4. Fully fill the order (partial → filled).
	filledAt := time.Now().UTC().Truncate(time.Microsecond)
	order.Status = domain.OrderStatusFilled
	order.FilledQuantity = 10
	finalAvgPrice := 185.75
	order.FilledAvgPrice = &finalAvgPrice
	order.FilledAt = &filledAt
	if err := r.Order.Update(ctx, order); err != nil {
		t.Fatalf("Update() to filled: %v", err)
	}

	// Record second fill trade.
	trade2 := &domain.Trade{
		OrderID:    &order.ID,
		PositionID: &position.ID,
		Ticker:     "AAPL",
		Side:       domain.OrderSideBuy,
		Quantity:   5,
		Price:      186.00,
		Fee:        0.30,
		ExecutedAt: time.Now().UTC().Truncate(time.Microsecond),
	}
	if err := r.Trade.Create(ctx, trade2); err != nil {
		t.Fatalf("Create() trade2: %v", err)
	}

	// Verify order is fully filled.
	filled, err := r.Order.Get(ctx, order.ID)
	if err != nil {
		t.Fatalf("Get() after fill: %v", err)
	}
	if filled.Status != domain.OrderStatusFilled {
		t.Fatalf("expected filled, got %q", filled.Status)
	}
	if filled.FilledQuantity != 10 {
		t.Fatalf("expected FilledQuantity=10, got %v", filled.FilledQuantity)
	}

	// Verify trades linked to the order.
	trades, err := r.Trade.GetByOrder(ctx, order.ID, repository.TradeFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("GetByOrder(): %v", err)
	}
	if len(trades) != 2 {
		t.Fatalf("expected 2 trades for order, got %d", len(trades))
	}

	// 5. Update position with fill data.
	position.Quantity = 10
	position.AvgEntry = 185.75
	currentPrice := 190.00
	unrealizedPnL := (190.00 - 185.75) * 10
	position.CurrentPrice = &currentPrice
	position.UnrealizedPnL = &unrealizedPnL
	if err := r.Position.Update(ctx, position); err != nil {
		t.Fatalf("Update() position: %v", err)
	}

	updatedPos, err := r.Position.Get(ctx, position.ID)
	if err != nil {
		t.Fatalf("Get() position: %v", err)
	}
	if updatedPos.Quantity != 10 {
		t.Fatalf("expected position Quantity=10, got %v", updatedPos.Quantity)
	}
	if updatedPos.CurrentPrice == nil || *updatedPos.CurrentPrice != currentPrice {
		t.Fatalf("expected CurrentPrice=%.2f, got %v", currentPrice, updatedPos.CurrentPrice)
	}
	if updatedPos.UnrealizedPnL == nil || *updatedPos.UnrealizedPnL != unrealizedPnL {
		t.Fatalf("expected UnrealizedPnL=%.2f, got %v", unrealizedPnL, updatedPos.UnrealizedPnL)
	}

	// Verify trades are linked to the position.
	posTrades, err := r.Trade.GetByPosition(ctx, position.ID, repository.TradeFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("GetByPosition(): %v", err)
	}
	if len(posTrades) != 2 {
		t.Fatalf("expected 2 trades for position, got %d", len(posTrades))
	}

	// 6. Close position.
	closedAt := time.Now().UTC().Truncate(time.Microsecond)
	position.RealizedPnL = 42.50
	position.UnrealizedPnL = nil
	position.ClosedAt = &closedAt
	if err := r.Position.Update(ctx, position); err != nil {
		t.Fatalf("Update() close position: %v", err)
	}

	closedPos, err := r.Position.Get(ctx, position.ID)
	if err != nil {
		t.Fatalf("Get() closed position: %v", err)
	}
	if closedPos.ClosedAt == nil {
		t.Fatal("expected ClosedAt to be set")
	}
	if closedPos.RealizedPnL != 42.50 {
		t.Fatalf("expected RealizedPnL=42.50, got %v", closedPos.RealizedPnL)
	}
	if closedPos.UnrealizedPnL != nil {
		t.Fatalf("expected nil UnrealizedPnL after close, got %v", closedPos.UnrealizedPnL)
	}

	// 7. Verify open positions no longer include the closed one.
	open, err := r.Position.GetOpen(ctx, repository.PositionFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("GetOpen(): %v", err)
	}
	for _, p := range open {
		if p.ID == position.ID {
			t.Fatal("closed position should not appear in GetOpen()")
		}
	}
}

// TestIntegration_OrderLifecycle_CancelledOrder verifies that an order
// can transition from submitted to cancelled.
func TestIntegration_OrderLifecycle_CancelledOrder(t *testing.T) {
	db := newTestDB(t)
	r := newRepos(db)
	ctx := context.Background()

	strategy := createStrategy(t, ctx, r.Strategy, "Cancel Test", "MSFT")
	order := createOrder(t, ctx, r.Order, strategy.ID, nil, "MSFT", domain.OrderSideBuy, domain.OrderTypeMarket, 5)

	// Submit.
	submittedAt := time.Now().UTC().Truncate(time.Microsecond)
	order.Status = domain.OrderStatusSubmitted
	order.SubmittedAt = &submittedAt
	if err := r.Order.Update(ctx, order); err != nil {
		t.Fatalf("Update() to submitted: %v", err)
	}

	// Cancel.
	order.Status = domain.OrderStatusCancelled
	if err := r.Order.Update(ctx, order); err != nil {
		t.Fatalf("Update() to cancelled: %v", err)
	}

	got, err := r.Order.Get(ctx, order.ID)
	if err != nil {
		t.Fatalf("Get() after cancel: %v", err)
	}
	if got.Status != domain.OrderStatusCancelled {
		t.Fatalf("expected cancelled, got %q", got.Status)
	}
}

// TestIntegration_OrderLifecycle_MultipleStrategies verifies order
// isolation between strategies and pipeline runs.
func TestIntegration_OrderLifecycle_MultipleStrategies(t *testing.T) {
	db := newTestDB(t)
	r := newRepos(db)
	ctx := context.Background()

	stratA := createStrategy(t, ctx, r.Strategy, "Strategy A", "AAPL")
	stratB := createStrategy(t, ctx, r.Strategy, "Strategy B", "MSFT")

	runA := uuid.New()
	runB := uuid.New()

	createOrder(t, ctx, r.Order, stratA.ID, &runA, "AAPL", domain.OrderSideBuy, domain.OrderTypeMarket, 10)
	createOrder(t, ctx, r.Order, stratA.ID, &runA, "AAPL", domain.OrderSideSell, domain.OrderTypeLimit, 5)
	createOrder(t, ctx, r.Order, stratB.ID, &runB, "MSFT", domain.OrderSideBuy, domain.OrderTypeMarket, 20)

	// Verify strategy scoping.
	ordersA, err := r.Order.GetByStrategy(ctx, stratA.ID, repository.OrderFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("GetByStrategy(A): %v", err)
	}
	if len(ordersA) != 2 {
		t.Fatalf("expected 2 orders for strategy A, got %d", len(ordersA))
	}

	ordersB, err := r.Order.GetByStrategy(ctx, stratB.ID, repository.OrderFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("GetByStrategy(B): %v", err)
	}
	if len(ordersB) != 1 {
		t.Fatalf("expected 1 order for strategy B, got %d", len(ordersB))
	}

	// Verify run scoping.
	runOrders, err := r.Order.GetByRun(ctx, runA, repository.OrderFilter{}, 10, 0)
	if err != nil {
		t.Fatalf("GetByRun(A): %v", err)
	}
	if len(runOrders) != 2 {
		t.Fatalf("expected 2 orders for run A, got %d", len(runOrders))
	}
}
