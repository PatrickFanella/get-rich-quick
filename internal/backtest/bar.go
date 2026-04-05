package backtest

import "github.com/PatrickFanella/get-rich-quick/internal/domain"

// Bar is a convenience alias for domain.OHLCV used by the automation and
// discovery packages. The full automation package will flesh this out further.
type Bar = domain.OHLCV
