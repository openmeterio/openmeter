package streaming

import (
	"time"

	"github.com/openmeterio/openmeter/openmeter/meter"
)

const (
	// MinWindowSizeDuration is the minimum window size the aggregation can represent.
	MinWindowSizeDuration = time.Second
	// MinWindowSize is the minimum window size the aggregation can represent.
	MinWindowSize = meter.WindowSizeSecond
)
