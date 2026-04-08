package signal

import (
	"context"
	"time"
)

// RawSignalEvent is an unprocessed event emitted by a signal source.
type RawSignalEvent struct {
	Source     string
	Title      string
	Body       string
	URL        string
	Metadata   map[string]any
	ReceivedAt time.Time
}

// SignalSource is implemented by all signal source adapters.
type SignalSource interface {
	// Name returns a stable, human-readable identifier for this source.
	Name() string
	// Start begins streaming events. The returned channel is closed when ctx is cancelled.
	Start(ctx context.Context) (<-chan RawSignalEvent, error)
}
