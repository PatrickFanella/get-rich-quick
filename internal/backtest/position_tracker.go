package backtest

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/PatrickFanella/get-rich-quick/internal/domain"
)

// TrackedPosition is a backtest position snapshot with derived valuation fields.
type TrackedPosition struct {
	Ticker        string
	Side          domain.PositionSide
	Quantity      float64
	AvgEntry      float64
	CostBasis     float64
	CurrentPrice  float64
	MarketValue   float64
	UnrealizedPnL float64
	RealizedPnL   float64
	OpenedAt      time.Time
}

// EquityPoint captures the portfolio state at a backtest bar.
type EquityPoint struct {
	Timestamp     time.Time
	Cash          float64
	MarketValue   float64
	Equity        float64
	RealizedPnL   float64
	UnrealizedPnL float64
	TotalPnL      float64
}

// PositionTracker tracks backtest positions, P&L, and the equity curve.
type PositionTracker struct {
	initialCash float64
	cash        float64
	realizedPnL float64
	positions   map[string]*domain.Position
	equityCurve []EquityPoint
}

// NewPositionTracker constructs a tracker seeded with an initial cash balance.
func NewPositionTracker(initialCash float64) (*PositionTracker, error) {
	if initialCash < 0 {
		return nil, errors.New("backtest: initial cash must be greater than or equal to zero")
	}

	return &PositionTracker{
		initialCash: initialCash,
		cash:        initialCash,
		positions:   make(map[string]*domain.Position),
	}, nil
}

// ApplyTrade updates positions and cash based on an executed simulated fill.
func (t *PositionTracker) ApplyTrade(trade domain.Trade) error {
	if t == nil {
		return errors.New("backtest: position tracker is required")
	}

	ticker, err := normalizeTrackerTicker(trade.Ticker)
	if err != nil {
		return err
	}
	if !trade.Side.IsValid() {
		return fmt.Errorf("backtest: invalid order side %q", trade.Side)
	}
	if trade.Quantity <= 0 {
		return errors.New("backtest: trade quantity must be greater than zero")
	}
	if trade.Price <= 0 {
		return errors.New("backtest: trade price must be greater than zero")
	}
	if trade.Fee < 0 {
		return errors.New("backtest: trade fee must be greater than or equal to zero")
	}

	notional := trade.Price * trade.Quantity
	if trade.Side == domain.OrderSideBuy {
		t.cash -= notional + trade.Fee
	} else {
		t.cash += notional - trade.Fee
	}

	fillSide := trackerPositionSide(trade.Side)
	position, ok := t.positions[ticker]
	if !ok {
		t.positions[ticker] = newTrackedPosition(ticker, fillSide, trade.Quantity, trade.Price, trade.Fee, trade.ExecutedAt)
		t.refreshPositionMetrics(t.positions[ticker], trade.Price)
		return nil
	}

	if position.Side == fillSide {
		totalQuantity := position.Quantity + trade.Quantity
		totalCostBasis := position.AvgEntry*position.Quantity + openingCostBasis(fillSide, trade.Price, trade.Quantity, trade.Fee)
		position.Quantity = totalQuantity
		position.AvgEntry = totalCostBasis / totalQuantity
		t.refreshPositionMetrics(position, trade.Price)
		return nil
	}

	closedQuantity := math.Min(position.Quantity, trade.Quantity)
	closingFee := trade.Fee * (closedQuantity / trade.Quantity)
	realized := closingPnL(position.Side, position.AvgEntry, trade.Price, closedQuantity, closingFee)
	position.RealizedPnL += realized
	t.realizedPnL += realized

	if trade.Quantity < position.Quantity {
		position.Quantity -= trade.Quantity
		t.refreshPositionMetrics(position, trade.Price)
		return nil
	}

	carryOverRealized := position.RealizedPnL
	delete(t.positions, ticker)

	remainingQuantity := trade.Quantity - closedQuantity
	if remainingQuantity <= 0 {
		return nil
	}

	openingFee := trade.Fee - closingFee
	t.positions[ticker] = newTrackedPosition(ticker, fillSide, remainingQuantity, trade.Price, openingFee, trade.ExecutedAt)
	t.positions[ticker].RealizedPnL = carryOverRealized
	t.refreshPositionMetrics(t.positions[ticker], trade.Price)
	return nil
}

// UpdateMarketPrice refreshes a ticker's mark price for unrealized P&L valuation.
func (t *PositionTracker) UpdateMarketPrice(ticker string, price float64) error {
	if t == nil {
		return errors.New("backtest: position tracker is required")
	}
	if price <= 0 {
		return errors.New("backtest: market price must be greater than zero")
	}

	normalized, err := normalizeTrackerTicker(ticker)
	if err != nil {
		return err
	}
	position, ok := t.positions[normalized]
	if !ok {
		return nil
	}

	t.refreshPositionMetrics(position, price)
	return nil
}

// RecordEquity appends and returns an equity-curve point using current mark prices.
func (t *PositionTracker) RecordEquity(timestamp time.Time) EquityPoint {
	point := EquityPoint{
		Timestamp:   timestamp,
		Cash:        t.cash,
		RealizedPnL: t.realizedPnL,
	}

	for _, position := range t.positions {
		price := trackerCurrentPrice(position)
		point.MarketValue += price * position.Quantity
		point.UnrealizedPnL += trackerUnrealizedPnL(position, price)
		if position.Side == domain.PositionSideLong {
			point.Equity += price * position.Quantity
			continue
		}
		point.Equity -= price * position.Quantity
	}

	point.Equity += t.cash
	point.TotalPnL = point.Equity - t.initialCash
	t.equityCurve = append(t.equityCurve, point)

	return point
}

// Positions returns defensive copies of open tracked positions sorted by ticker.
func (t *PositionTracker) Positions() []TrackedPosition {
	if t == nil {
		return nil
	}

	tickers := make([]string, 0, len(t.positions))
	for ticker := range t.positions {
		tickers = append(tickers, ticker)
	}
	sort.Strings(tickers)

	positions := make([]TrackedPosition, 0, len(tickers))
	for _, ticker := range tickers {
		position := t.positions[ticker]
		price := trackerCurrentPrice(position)
		positions = append(positions, TrackedPosition{
			Ticker:        position.Ticker,
			Side:          position.Side,
			Quantity:      position.Quantity,
			AvgEntry:      position.AvgEntry,
			CostBasis:     position.AvgEntry * position.Quantity,
			CurrentPrice:  price,
			MarketValue:   price * position.Quantity,
			UnrealizedPnL: trackerUnrealizedPnL(position, price),
			RealizedPnL:   position.RealizedPnL,
			OpenedAt:      position.OpenedAt,
		})
	}

	return positions
}

// EquityCurve returns a defensive copy of recorded equity points.
func (t *PositionTracker) EquityCurve() []EquityPoint {
	if t == nil {
		return nil
	}

	curve := make([]EquityPoint, len(t.equityCurve))
	copy(curve, t.equityCurve)
	return curve
}

func (t *PositionTracker) refreshPositionMetrics(position *domain.Position, price float64) {
	if position == nil {
		return
	}

	position.CurrentPrice = trackerFloatPtr(price)
	unrealized := trackerUnrealizedPnL(position, price)
	position.UnrealizedPnL = trackerFloatPtr(unrealized)
}

func newTrackedPosition(ticker string, side domain.PositionSide, quantity float64, price float64, fee float64, openedAt time.Time) *domain.Position {
	return &domain.Position{
		Ticker:     ticker,
		Side:       side,
		Quantity:   quantity,
		AvgEntry:   openingCostBasis(side, price, quantity, fee) / quantity,
		OpenedAt:   openedAt,
		ClosedAt:   nil,
		StopLoss:   nil,
		TakeProfit: nil,
	}
}

func normalizeTrackerTicker(ticker string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(ticker))
	if normalized == "" {
		return "", errors.New("backtest: trade ticker is required")
	}
	return normalized, nil
}

func trackerPositionSide(side domain.OrderSide) domain.PositionSide {
	if side == domain.OrderSideSell {
		return domain.PositionSideShort
	}
	return domain.PositionSideLong
}

func openingCostBasis(side domain.PositionSide, price float64, quantity float64, fee float64) float64 {
	if side == domain.PositionSideShort {
		return (price * quantity) - fee
	}
	return (price * quantity) + fee
}

func closingPnL(side domain.PositionSide, avgEntry float64, fillPrice float64, quantity float64, fee float64) float64 {
	if side == domain.PositionSideShort {
		return (avgEntry * quantity) - (fillPrice * quantity) - fee
	}
	return (fillPrice * quantity) - fee - (avgEntry * quantity)
}

func trackerCurrentPrice(position *domain.Position) float64 {
	if position.CurrentPrice != nil && *position.CurrentPrice > 0 {
		return *position.CurrentPrice
	}
	return position.AvgEntry
}

func trackerUnrealizedPnL(position *domain.Position, price float64) float64 {
	if position.Side == domain.PositionSideShort {
		return (position.AvgEntry - price) * position.Quantity
	}
	return (price - position.AvgEntry) * position.Quantity
}

func trackerFloatPtr(value float64) *float64 {
	return &value
}
