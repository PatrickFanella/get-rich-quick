package domain

import (
	"time"

	"github.com/google/uuid"
)

// PositionSide represents whether a position is long or short.
type PositionSide string

const (
	PositionSideLong  PositionSide = "long"
	PositionSideShort PositionSide = "short"
)

// String returns the string representation of a PositionSide.
func (s PositionSide) String() string {
	return string(s)
}

// Position represents an open or closed trading position.
type Position struct {
	ID            uuid.UUID    `json:"id"`
	StrategyID    *uuid.UUID   `json:"strategy_id,omitempty"`
	Ticker        string       `json:"ticker"`
	Side          PositionSide `json:"side"`
	Quantity      float64      `json:"quantity"`
	AvgEntry      float64      `json:"avg_entry"`
	CurrentPrice  *float64     `json:"current_price,omitempty"`
	UnrealizedPnL *float64     `json:"unrealized_pnl,omitempty"`
	RealizedPnL   float64      `json:"realized_pnl"`
	StopLoss      *float64     `json:"stop_loss,omitempty"`
	TakeProfit    *float64     `json:"take_profit,omitempty"`
	OpenedAt      time.Time    `json:"opened_at"`
	ClosedAt      *time.Time   `json:"closed_at,omitempty"`
}
